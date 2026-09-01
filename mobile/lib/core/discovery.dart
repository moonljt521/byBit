import 'dart:async';
import 'dart:convert';
import 'dart:io';

/// 局域网自动发现后端服务器。
///
/// 这里修掉了三个会导致真机「搜不到 / 搜到错的」的真实缺陷：
///
/// 1. **硬编码 /24 网段**：原实现取本机 IP 前三个字节当网段，只扫 254 个地址。
///    但办公网 / 校园网常见 /23、/22 掩码（例如 10.1.70.207/23 的网段是
///    10.1.70.0 ~ 10.1.71.255，共 512 个）。手机一旦落在 10.1.71.x，
///    扫描范围就变成 10.1.71.1~254，而电脑在 10.1.70.207 —— 永远扫不到。
///    → 现在按 /22 对齐展开最多 4 个 /24 块，完整覆盖 /24、/23、/22 掩码。
///
/// 2. **取错本机 IP**：原实现返回第一个非回环 IPv4，开 VPN（utun/tun）或
///    蜂窝数据时优先命中隧道接口（常见 198.18.x、100.64.x），网段直接算错。
///    → 现在排除隧道 / CGNAT / 链路本地地址，并优先取 Wi-Fi 接口。
///
/// 3. **只做 TCP 连通探测**：原实现只要 8080 端口能连上就认定是服务器，
///    打印机、路由器、其他开发机都会被误判并直接返回。
///    → 现在先用 TCP 轻量预筛，命中后再做一次真实
///    HTTP GET /api/v1/health 校验响应体，确认是 CryptoSim 才算数。
class ServerDiscovery {
  ServerDiscovery._();
  static final ServerDiscovery I = ServerDiscovery._();

  static const int port = 8080;

  /// 单批并发探测数。Wi-Fi 下不存在的 IP 通常在 ARP 阶段就快速失败，
  /// 600ms 超时只对被防火墙静默丢弃的地址生效。
  /// 命中在自身网段时约 1 秒；最坏情况（需扫满 4 个块）约 7 秒。
  static const int _batchSize = 96;

  /// 一次发现最多扫描的网段数，防止多网卡环境扫太久。
  static const int _maxPrefixes = 4;

  Future<String?> discover({
    Duration connectTimeout = const Duration(milliseconds: 600),
    void Function(int scanned, int total)? onProgress,
  }) async {
    final prefixes = await candidatePrefixes();
    if (prefixes.isEmpty) return null;

    final total = prefixes.length * 254;
    var scanned = 0;

    for (final prefix in prefixes) {
      for (var start = 1; start <= 254; start += _batchSize) {
        final end = (start + _batchSize - 1).clamp(1, 254);
        final results = await Future.wait([
          for (var i = start; i <= end; i++) _probe('$prefix.$i', connectTimeout),
        ]);
        scanned += end - start + 1;
        onProgress?.call(scanned, total);

        final hits = results.whereType<String>().toList();
        // 同一批多个命中时按字典序取第一个，保证多次扫描结果稳定（不随机漂移）
        if (hits.isNotEmpty) {
          hits.sort();
          return hits.first;
        }
      }
    }
    return null;
  }

  /// 本机在局域网内的所有候选 IPv4（已排除隧道 / CGNAT / 链路本地），Wi-Fi 优先。
  /// 供 UI 在扫描失败时展示，方便人工比对网段。
  Future<List<String>> localIps() async {
    final interfaces = await NetworkInterface.list(
      type: InternetAddressType.IPv4,
      includeLoopback: false,
    );
    final wifi = <String>[];
    final others = <String>[];
    for (final i in interfaces) {
      final isWifi = _isWifiInterface(i.name);
      for (final a in i.addresses) {
        if (a.isLoopback || _isUnusable(a.address)) continue;
        (isWifi ? wifi : others).add(a.address);
      }
    }
    return <String>[...wifi, ...others];
  }

  /// 派生待扫描网段：自身 /24 排最前，其余按 /22 对齐展开并按距离排序。
  /// 对外暴露，便于 UI 直接展示「准备扫哪些网段」，排查网段算错时一目了然。
  Future<List<String>> candidatePrefixes() async {
    final prefixes = <String>[];
    for (final ip in await localIps()) {
      for (final p in _prefixesFor(ip)) {
        if (!prefixes.contains(p)) prefixes.add(p);
      }
      if (prefixes.length >= _maxPrefixes) break;
    }
    return prefixes;
  }

  /// 由单个 IP 派生候选网段，按「离自身多近」排序，近的先扫。
  ///
  /// 掩码比 /24 大时只扫自身 /24 会漏掉同网段的其余地址 —— 这是真机搜不到的主因：
  /// - /23 → 2 个块（10.1.70.207/23 的网段是 10.1.70.0~10.1.71.255）
  /// - /22 → 4 个块
  ///
  /// 这里按 /22 对齐展开最多 4 个块，完整覆盖 /24、/23、/22 三种常见掩码；
  /// 自身所在块排在最前，命中即返回，所以小网段下耗时和原来一样是秒级。
  static List<String> _prefixesFor(String ip) {
    final parts = ip.split('.');
    if (parts.length != 4) return const [];
    final nums = parts.map(int.tryParse).toList();
    if (nums.any((e) => e == null)) return const [];

    final a = nums[0]!;
    final b = nums[1]!;
    final c = nums[2]!;

    final base = c & 0xFC; // /22 对齐
    final blocks = <int>[
      for (var i = 0; i < 4; i++)
        if (base + i <= 255) base + i,
    ];
    blocks.sort((x, y) => (x - c).abs().compareTo((y - c).abs()));
    return blocks.map((e) => '$a.$b.$e').toList();
  }

  static bool _isWifiInterface(String name) {
    final n = name.toLowerCase();
    return n.startsWith('wlan') || n.startsWith('en') || n.contains('wifi');
  }

  /// 排除拿不到局域网服务器的地址：
  /// - 169.254/16  链路本地，说明没拿到 DHCP
  /// - 100.64/10   运营商级 NAT，蜂窝数据常见
  /// - 198.18/15   VPN 隧道（Clash / Surge / WireGuard 等）常见段
  static bool _isUnusable(String ip) {
    final parts = ip.split('.');
    if (parts.length != 4) return false;
    if (parts[0] == '169' && parts[1] == '254') return true; // 链路本地
    if (parts[0] == '198' && (parts[1] == '18' || parts[1] == '19')) return true; // VPN 隧道
    if (parts[0] == '100') {
      final second = int.tryParse(parts[1]);
      if (second != null && second >= 64 && second <= 127) return true; // CGNAT
    }
    return false;
  }

  /// 探测单个地址：先 TCP 预筛（轻量，覆盖 99% 的失败场景），
  /// 端口通了再用 HTTP 校验身份，避免把打印机 / 路由器当成后端。
  Future<String?> _probe(String ip, Duration timeout) async {
    Socket? socket;
    try {
      socket = await Socket.connect(ip, port, timeout: timeout);
    } catch (_) {
      return null;
    } finally {
      socket?.destroy();
    }
    return _verify(ip, port, timeout);
  }

  /// 校验一个手填地址是否真是可用的 CryptoSim 后端，供「保存」时即时反馈。
  /// 直接在 baseUrl 后面拼 /health，因此端口和路径前缀都按用户填的来。
  Future<bool> verifyUrl(
    String baseUrl, {
    Duration timeout = const Duration(seconds: 3),
  }) async {
    final uri = Uri.tryParse(baseUrl);
    if (uri == null || uri.host.isEmpty) return false;
    if (uri.scheme != 'http' && uri.scheme != 'https') return false;

    final health = uri.replace(
      path: '${uri.path}/health',
      query: null,
      fragment: null,
    );
    final client = HttpClient()..connectionTimeout = timeout;
    try {
      final req = await client.getUrl(health).timeout(timeout);
      final resp = await req.close().timeout(timeout);
      if (resp.statusCode != 200) return false;
      final body = await resp.transform(utf8.decoder).join().timeout(timeout);
      return body.contains('"status"');
    } catch (_) {
      return false;
    } finally {
      client.close(force: true);
    }
  }

  /// 校验对端确实是 CryptoSim 后端：GET /api/v1/health 需返回 200 且含状态字段。
  static Future<String?> _verify(String host, int port, Duration timeout) async {
    final client = HttpClient()..connectionTimeout = timeout;
    try {
      final req = await client
          .getUrl(Uri.parse('http://$host:$port/api/v1/health'))
          .timeout(timeout);
      final resp = await req.close().timeout(timeout);
      if (resp.statusCode != 200) return null;
      final body = await resp.transform(utf8.decoder).join().timeout(timeout);
      if (!body.contains('"status"')) return null;
      return 'http://$host:$port/api/v1';
    } catch (_) {
      return null;
    } finally {
      client.close(force: true);
    }
  }
}
