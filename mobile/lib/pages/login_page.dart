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

  @override
  void dispose() {
    _account.dispose();
    _password.dispose();
    _email.dispose();
    _username.dispose();
    super.dispose();
  }

  Future<void> _editServer() async {
    final ctrl = TextEditingController(text: CredentialStore.I.baseUrl);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('服务器地址'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: ctrl, decoration: const InputDecoration(hintText: 'API 地址')),
            const SizedBox(height: 8),
            OutlinedButton.icon(
              icon: const Icon(Icons.search),
              label: const Text('自动搜索局域网服务器'),
              onPressed: () async {
                ScaffoldMessenger.of(ctx).showSnackBar(
                  const SnackBar(content: Text('正在扫描局域网（约 3-5 秒）…')),
                );
                final found = await ServerDiscovery.I.discover();
                if (found != null) {
                  ctrl.text = found;
                  if (ctx.mounted) {
                    ScaffoldMessenger.of(ctx)
                        .showSnackBar(SnackBar(content: Text('已找到：$found')));
                  }
                } else if (ctx.mounted) {
                  ScaffoldMessenger.of(ctx).showSnackBar(const SnackBar(
                      content: Text('未找到服务器：确认电脑已启动后端且手机连同一 Wi-Fi')));
                }
              },
            ),
            const SizedBox(height: 8),
            const Text(
              '真机请填电脑局域网 IP，如 http://192.168.3.37:8080/api/v1\n'
              '安卓模拟器可用 http://10.0.2.2:8080/api/v1',
              style: TextStyle(fontSize: 12),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok == true) {
      await CredentialStore.I.setBaseUrl(ctrl.text.trim());
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text('已保存：${CredentialStore.I.baseUrl}'),
        ));
      }
    }
    ctrl.dispose();
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
            tooltip: '服务器地址',
            icon: const Icon(Icons.dns, color: Colors.white54),
            onPressed: _editServer,
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
