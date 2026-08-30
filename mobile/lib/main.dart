import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:hive_flutter/hive_flutter.dart';

import 'core/api_client.dart';
import 'core/theme.dart';
import 'providers/app_providers.dart';
import 'router.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
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
