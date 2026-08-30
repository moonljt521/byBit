import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../core/api_client.dart';
import '../core/theme.dart';
import '../providers/app_providers.dart';

/// 我的：账号、服务器地址、重置 API 凭证、退出登录。
class MinePage extends ConsumerStatefulWidget {
  const MinePage({super.key});

  @override
  ConsumerState<MinePage> createState() => _MinePageState();
}

class _MinePageState extends ConsumerState<MinePage> {
  late final _url = TextEditingController(text: CredentialStore.I.baseUrl);
  final _oldSecret = TextEditingController();

  @override
  void dispose() {
    _url.dispose();
    _oldSecret.dispose();
    super.dispose();
  }

  void _snack(String msg, {bool error = false}) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), backgroundColor: error ? AppTheme.down : null),
    );
  }

  Future<void> _saveUrl() async {
    await CredentialStore.I.setBaseUrl(_url.text.trim());
    if (mounted) _snack('服务器地址已保存：${CredentialStore.I.baseUrl}');
  }

  Future<void> _resetCredentials() async {
    try {
      await ApiClient.I.resetCredentials();
      if (mounted) _snack('API 凭证已轮换（旧签名立即失效）');
    } on ApiException catch (e) {
      _snack(e.message, error: true);
    }
  }

  Future<void> _logout() async {
    await ref.read(authProvider.notifier).logout();
    if (mounted) context.go('/login');
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: ListView(
        children: [
          const SizedBox(height: 12),
          ListTile(
            leading: const CircleAvatar(child: Icon(Icons.person)),
            title: Text(auth.username.isEmpty ? '未登录' : auth.username,
                style: const TextStyle(fontWeight: FontWeight.bold)),
            subtitle:
                const Text('虚拟资金账户 · HMAC 验签已启用', style: TextStyle(fontSize: 12)),
          ),
          const Divider(),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Text('服务器地址', style: Theme.of(context).textTheme.titleSmall),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                TextField(
                  controller: _url,
                  decoration: const InputDecoration(
                    hintText: 'API 地址（安卓模拟器: http://10.0.2.2:8080/api/v1）',
                  ),
                ),
                const SizedBox(height: 8),
                OutlinedButton(onPressed: _saveUrl, child: const Text('保存地址')),
                const SizedBox(height: 4),
                Text(
                  '真机调试请填电脑局域网 IP；iOS 模拟器可用 http://127.0.0.1:8080/api/v1',
                  style: TextStyle(fontSize: 12, color: Colors.grey.shade600),
                ),
              ],
            ),
          ),
          const Divider(height: 32),
          Consumer(builder: (context, ref, _) {
            final lang = ref.watch(langProvider);
            return ListTile(
              leading: const Icon(Icons.language),
              title: const Text('语言 / Language'),
              trailing: FilledButton.tonal(
                onPressed: () => ref.read(langProvider.notifier).toggle(),
                child: Text(lang.code == 'zh' ? '切换到 English' : 'Switch to 中文'),
              ),
            );
          }),
          ListTile(
            leading: const Icon(Icons.key),
            title: const Text('重置 API 凭证'),
            subtitle: const Text('轮换 HMAC 密钥对，旧签名立即失效'),
            onTap: _resetCredentials,
          ),
          ListTile(
            leading: const Icon(Icons.logout, color: AppTheme.down),
            title: const Text('退出登录', style: TextStyle(color: AppTheme.down)),
            onTap: _logout,
          ),
          const Padding(
            padding: EdgeInsets.all(16),
            child: Text(
              'CryptoSim v0.2.0\n本应用为学习用途的模拟交易所：行情来自 OKX/Binance 公开接口，'
              '请求使用 JWT + HMAC-SHA256 双重校验（防篡改/防重放），'
              '凭证经 Android Keystore / iOS Keychain 加密存储，'
              '所有资金均为虚拟资金，不涉及任何真实充值提币。',
              style: TextStyle(fontSize: 12, color: Colors.grey),
            ),
          ),
        ],
      ),
    );
  }
}
