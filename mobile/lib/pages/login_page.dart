import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../core/api_client.dart';
import '../core/discovery.dart';
import '../core/theme.dart';
import '../providers/app_providers.dart';

/// 登录 / 注册页。
class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _account = TextEditingController();
  final _password = TextEditingController();
  final _email = TextEditingController();
  final _username = TextEditingController();
  bool _registerMode = false;
  bool _busy = false;

  /// 后台自愈扫描进行中（换网段 / 后端重启换 IP 后地址会失效）。
  bool _probing = false;

  @override
  void initState() {
    super.initState();
    // 地址不许手填，所以失效时只能靠扫描自愈，进页面先试一次
    WidgetsBinding.instance.addPostFrameCallback((_) => _autoDiscoverIfNeeded());
  }

  @override
  void dispose() {
    _account.dispose();
    _password.dispose();
    _email.dispose();
    _username.dispose();
    super.dispose();
  }

  /// 当前地址不通时静默扫一次，成功即应用；失败不打断，留给面板手动触发。
  Future<void> _autoDiscoverIfNeeded() async {
    if (!mounted) return;
    setState(() => _probing = true);
    try {
      if (await ServerDiscovery.I.verifyUrl(CredentialStore.I.baseUrl)) return;
      final found = await ServerDiscovery.I.discover();
      if (found != null && mounted) {
        await CredentialStore.I.setBaseUrl(found);
        if (mounted) _snack('已自动发现服务器：$found');
      }
    } finally {
      if (mounted) setState(() => _probing = false);
    }
  }

  /// 打开服务器发现面板。地址只由扫描结果决定，面板内不可编辑。
  Future<void> _discoverDialog() async {
    await showDialog(
      context: context,
      builder: (_) => const _DiscoverPanel(),
    );
  }

  Future<void> _submit() async {
    setState(() => _busy = true);
    try {
      if (_registerMode) {
        await ApiClient.I.register(_email.text.trim(), _username.text.trim(), _password.text);
        if (mounted) _snack('注册成功，已赠送 10,000 虚拟 USDT');
      } else {
        await ApiClient.I.login(_account.text.trim(), _password.text);
      }
      ref.read(authProvider.notifier).refresh();
      if (mounted) context.go('/markets');
    } on ApiException catch (e) {
      if (mounted) _snack(e.message, error: true);
    } catch (e) {
      if (mounted) _snack('网络异常：$e', error: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _snack(String msg, {bool error = false}) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: error ? AppTheme.down : null,
    ));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.bg,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        actions: [
          IconButton(
            tooltip: '服务器发现',
            icon: _probing
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.dns, color: Colors.white54),
            onPressed: _probing ? null : _discoverDialog,
          ),
        ],
      ),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(28),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text('CryptoSim',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                      fontSize: 36, fontWeight: FontWeight.bold, color: AppTheme.accent)),
              const SizedBox(height: 6),
              Text('虚拟加密货币交易所 · 资金全虚拟 · 仅供学习',
                  textAlign: TextAlign.center,
                  style: TextStyle(fontSize: 12, color: Colors.grey.shade500)),
              const SizedBox(height: 32),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    children: [
                      if (_registerMode) ...[
                        TextField(
                          controller: _email,
                          keyboardType: TextInputType.emailAddress,
                          decoration: const InputDecoration(hintText: '邮箱', hintStyle: TextStyle(color: Color(0x38FFFFFF), fontSize: 13)),
                        ),
                        const SizedBox(height: 12),
                        TextField(
                          controller: _username,
                          decoration: const InputDecoration(
                              hintText: '用户名（3-20 位字母数字下划线）',
                              hintStyle: TextStyle(color: Color(0x38FFFFFF), fontSize: 13)),
                        ),
                        const SizedBox(height: 12),
                      ],
                      if (!_registerMode)
                        TextField(
                          controller: _account,
                          decoration: const InputDecoration(
                              hintText: '邮箱 / 用户名',
                              hintStyle: TextStyle(color: Color(0x38FFFFFF), fontSize: 13)),
                        ),
                      if (!_registerMode) const SizedBox(height: 12),
                      TextField(
                        controller: _password,
                        obscureText: true,
                        decoration: const InputDecoration(
                              hintText: '密码（至少 8 位）',
                              hintStyle: TextStyle(color: Color(0x38FFFFFF), fontSize: 13)),
                      ),
                      const SizedBox(height: 20),
                      FilledButton(
                        onPressed: _busy ? null : _submit,
                        style: FilledButton.styleFrom(
                            backgroundColor: AppTheme.accent,
                            minimumSize: const Size.fromHeight(50)),
                        child: Builder(builder: (context) {
                          final isEn =
                              ref.watch(langProvider).code == 'en';
                          return Text(_busy
                              ? '…'
                              : (_registerMode
                                  ? (isEn
                                      ? 'Sign up & get 10,000 USDT'
                                      : '注册并领取虚拟资金')
                                  : (isEn ? 'Sign in' : '登录')));
                        }),
                      ),
                      const SizedBox(height: 8),
                      TextButton(
                        onPressed: () =>
                            setState(() => _registerMode = !_registerMode),
                        child: Text(
                          _registerMode ? '已有账号？去登录' : '没有账号？立即注册（送 10,000 USDT）',
                          style: TextStyle(color: Colors.grey.shade400),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 服务器发现面板。
///
/// 只有「扫描 → 展示 → 应用」一条路径，不提供任何地址编辑入口：
/// 手填最容易把网段写错，而一旦写错就再也分不清是地址填错还是服务没起来。
/// 面板同时展示**本机 IP 与待扫描网段** —— 网段算错时一眼就能看出来。
class _DiscoverPanel extends StatefulWidget {
  const _DiscoverPanel();

  @override
  State<_DiscoverPanel> createState() => _DiscoverPanelState();
}

class _DiscoverPanelState extends State<_DiscoverPanel> {
  bool _scanning = false;
  bool _checking = true;
  int _scanned = 0;
  int _total = 0;
  bool? _connected;
  String _note = '';
  bool _noteOk = true;
  List<String> _ips = const [];
  List<String> _prefixes = const [];

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    final ips = await ServerDiscovery.I.localIps();
    final prefixes = await ServerDiscovery.I.candidatePrefixes();
    if (!mounted) return;
    setState(() {
      _ips = ips;
      _prefixes = prefixes;
    });
    await _recheck();
  }

  /// 探一遍当前地址，刷新连接状态灯。
  Future<void> _recheck() async {
    setState(() => _checking = true);
    final alive = await ServerDiscovery.I.verifyUrl(CredentialStore.I.baseUrl);
    if (!mounted) return;
    setState(() {
      _checking = false;
      _connected = alive;
    });
  }

  Future<void> _scan() async {
    if (_scanning) return;
    setState(() {
      _scanning = true;
      _scanned = 0;
      _total = 0;
      _note = '';
      _noteOk = true;
    });

    final sw = Stopwatch()..start();
    final found = await ServerDiscovery.I.discover(
      onProgress: (scanned, total) {
        if (!mounted) return;
        setState(() {
          _scanned = scanned;
          _total = total;
        });
      },
    );
    sw.stop();
    if (!mounted) return;

    if (found == null) {
      setState(() {
        _scanning = false;
        _noteOk = false;
        _note = _ips.isEmpty
            ? '未取到本机 IP，请确认已连上 Wi-Fi'
            : '未发现服务（耗时 ${(sw.elapsedMilliseconds / 1000).toStringAsFixed(1)}s）';
      });
      return;
    }

    await CredentialStore.I.setBaseUrl(found);
    await _recheck();
    if (!mounted) return;
    setState(() {
      _scanning = false;
      _noteOk = true;
      _note = '已连接 ${CredentialStore.I.baseUrl}'
          '（耗时 ${(sw.elapsedMilliseconds / 1000).toStringAsFixed(1)}s）';
    });
  }

  @override
  Widget build(BuildContext context) {
    final connected = _connected;
    return AlertDialog(
      backgroundColor: AppTheme.surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: const BorderSide(color: AppTheme.border),
      ),
      title: const Text('服务器发现', style: TextStyle(fontSize: 16)),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _kv('当前地址', CredentialStore.I.baseUrl, selectable: true),
            const SizedBox(height: 8),
            Row(children: [
              _dot(connected),
              const SizedBox(width: 6),
              Text(
                _checking
                    ? '检测中…'
                    : (connected == true ? '已连接' : '未连接'),
                style: const TextStyle(fontSize: 12, color: Colors.white54),
              ),
            ]),
            const Divider(height: 22),
            _kv('本机 IP', _ips.isEmpty ? '未取到' : _ips.join('、')),
            const SizedBox(height: 10),
            _kv('扫描网段',
                _prefixes.isEmpty ? '—' : _prefixes.map((p) => '$p.*').join('、')),
            if (_scanning) ...[
              const SizedBox(height: 16),
              LinearProgressIndicator(
                value: _total == 0 ? null : _scanned / _total,
                color: AppTheme.accent,
                backgroundColor: AppTheme.border,
              ),
              const SizedBox(height: 6),
              Text('已扫 $_scanned / $_total',
                  style: const TextStyle(fontSize: 11, color: Colors.white38)),
            ],
            if (_note.isNotEmpty) ...[
              const SizedBox(height: 12),
              Text(
                _note,
                style: TextStyle(
                    fontSize: 12,
                    color: _noteOk ? AppTheme.up : AppTheme.down),
              ),
            ],
            const SizedBox(height: 20),
            FilledButton.icon(
              onPressed: _scanning ? null : _scan,
              icon: _scanning
                  ? const SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.wifi_find, size: 18),
              label: Text(_scanning ? '扫描中…' : '搜索局域网服务器'),
              style: FilledButton.styleFrom(backgroundColor: AppTheme.accent),
            ),
            const SizedBox(height: 10),
            const Text(
              '地址由扫描结果决定，不可手动修改。\n'
              '需手机与电脑连同一 Wi-Fi，且电脑已启动后端。',
              style: TextStyle(fontSize: 11, color: Colors.white38, height: 1.5),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('关闭'),
        ),
      ],
    );
  }

  Widget _kv(String k, String v, {bool selectable = false}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(k, style: const TextStyle(fontSize: 11, color: Colors.white38)),
        const SizedBox(height: 3),
        selectable
            ? SelectableText(v,
                style: const TextStyle(fontSize: 12, fontFamily: 'monospace'))
            : Text(v, style: const TextStyle(fontSize: 12)),
      ],
    );
  }

  Widget _dot(bool? connected) {
    return Container(
      width: 7,
      height: 7,
      decoration: BoxDecoration(
        color: connected == null
            ? Colors.white24
            : (connected ? AppTheme.up : AppTheme.down),
        shape: BoxShape.circle,
      ),
    );
  }
}
