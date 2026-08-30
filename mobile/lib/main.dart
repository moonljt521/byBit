import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:hive_flutter/hive_flutter.dart';

import 'core/api_client.dart';
import 'core/theme.dart';
import 'providers/app_providers.dart';
import 'router.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  unawaited(Hive.initFlutter().then((_) => Hive.openBox('klines'))); // K线缓存懒预热，不阻塞启动
  // 全局兜底：build 阶段未捕获异常显示友好错误页（替代 Flutter 红屏）
  ErrorWidget.builder = (details) => Container(
        color: const Color(0xFF0B0E11),
        alignment: Alignment.center,
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.wifi_off, size: 56, color: Colors.white24),
            const SizedBox(height: 16),
            const Text('服务暂时不可用',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            Text('后端服务未启动或网络不可用，请检查后在当前页面下拉重试。',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 13, color: Colors.grey.shade500)),
          ],
        ),
      );
  await Hive.initFlutter();
  await Hive.openBox('klines'); // K 线本地缓存（离线可见）
  await CredentialStore.I.load();
  runApp(const ProviderScope(child: CryptoSimApp()));
}

class CryptoSimApp extends ConsumerStatefulWidget {
  const CryptoSimApp({super.key});

  @override
  ConsumerState<CryptoSimApp> createState() => _CryptoSimAppState();
}

class _CryptoSimAppState extends ConsumerState<CryptoSimApp> {
  @override
  void initState() {
    super.initState();
    // 启动时恢复本地凭证，再触发路由守卫计算
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await CredentialStore.I.load();
      await ref.read(langProvider.notifier).load();
      ref.read(authProvider.notifier).refresh();
    });
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authProvider);
    final router = buildRouter(ref);
    return MaterialApp.router(
      title: 'CryptoSim 模拟交易所',
      theme: AppTheme.dark(),
      routerConfig: router,
      builder: (context, child) {
        // 登录态变化时重建路由（守卫生效）
        return KeyedSubtree(key: ValueKey(auth.loggedIn), child: child!);
      },
    );
  }
}
