import 'dart:io';

/// 局域网自动发现后端：探测同网段所有 IP 的 8080 端口（/api/v1/health 语义）。
/// 手机自身 IP 推导网段（如 192.168.3.x），并发分批探测，命中即返回。
class ServerDiscovery {
  ServerDiscovery._();
  static final ServerDiscovery I = ServerDiscovery._();

  static const port = 8080;

  Future<String?> discover({
    Duration connectTimeout = const Duration(milliseconds: 600),
    void Function(int scanned, int total)? onProgress,
  }) async {
    final myIp = await _selfIp();
    if (myIp == null) return null;
    final prefix = myIp.split('.').sublist(0, 3).join('.');
    const total = 254;

    for (var start = 1; start <= total; start += 32) {
      final end = (start + 31).clamp(1, total);
      final results = await Future.wait([
        for (var i = start; i <= end; i++) _probe('$prefix.$i', connectTimeout),
      ]);
      onProgress?.call(end, total);
      final hits = results.whereType<String>().toList();
      if (hits.isNotEmpty) return hits.first;
    }
    return null;
  }

  Future<String?> _probe(String ip, Duration timeout) async {
    try {
      final s = await Socket.connect(ip, port, timeout: timeout);
      s.destroy();
      return 'http://$ip:$port/api/v1';
    } catch (_) {
      return null;
    }
  }

  Future<String?> _selfIp() async {
    final interfaces = await NetworkInterface.list(type: InternetAddressType.IPv4);
    for (final i in interfaces) {
      for (final a in i.addresses) {
        if (!a.isLoopback && !a.address.startsWith('169.254')) return a.address;
      }
    }
    return null;
  }
}
