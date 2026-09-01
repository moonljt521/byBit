import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/fallback_cache.dart';
import 'pages/assets_page.dart';
import 'pages/futures_page.dart';
import 'pages/learn_page.dart';
import 'pages/login_page.dart';
import 'pages/markets_page.dart';
import 'pages/mine_page.dart';
import 'pages/spot_page.dart';
import 'providers/app_providers.dart';

final rootNavigatorKey = GlobalKey<NavigatorState>();

class _Dest {
  final String path;
  final String zh;
  final String en;
  final IconData icon;
  final IconData selectedIcon;
  const _Dest(this.path, this.zh, this.en, this.icon, this.selectedIcon);
}

const _destinations = [
  _Dest('/markets', '行情', 'Markets', Icons.candlestick_chart_outlined, Icons.candlestick_chart),
  _Dest('/spot', '现货', 'Spot', Icons.swap_horiz_outlined, Icons.swap_horiz),
  _Dest('/futures', '合约', 'Futures', Icons.show_chart_outlined, Icons.show_chart),
  _Dest('/assets', '资产', 'Assets', Icons.account_balance_wallet_outlined, Icons.account_balance_wallet),
  _Dest('/learn', '学习', 'Learn', Icons.menu_book_outlined, Icons.menu_book),
  _Dest('/mine', '我的', 'Me', Icons.person_outline, Icons.person),
];

/// 底部 Tab 路由 + 登录守卫。
///
/// ⚠️ 路由实例整个 App 生命周期只应创建一次（见 main.dart 的 `_router ??=`），
/// 守卫必须用 `ref.read` **实时**读取登录态——闭包捕获创建时的快照的话，
/// 登录成功后守卫看到的还是未登录，会把 context.go('/markets') 静默弹回
/// /login，表现为「点登录无任何反应」。
GoRouter buildRouter(WidgetRef ref) {
  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation:
        ref.read(authProvider).loggedIn ? '/markets' : '/login',
    redirect: (context, state) {
      final auth = ref.read(authProvider); // 每次导航时实时读取
      final loggingIn = state.matchedLocation == '/login';
      if (!auth.loggedIn && !loggingIn) return '/login';
      if (auth.loggedIn && loggingIn) return '/markets';
      return null;
    },
    routes: [
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginPage(),
      ),
      ShellRoute(
        builder: (context, state, child) => HomeShell(child: child),
        routes: [
          GoRoute(path: '/markets', builder: (_, __) => const MarketsPage()),
          GoRoute(path: '/spot', builder: (_, __) => const SpotPage()),
          GoRoute(path: '/futures', builder: (_, __) => const FuturesPage()),
          GoRoute(path: '/assets', builder: (_, __) => const AssetsPage()),
          GoRoute(path: '/learn', builder: (_, __) => const LearnPage()),
          GoRoute(path: '/mine', builder: (_, __) => const MinePage()),
        ],
      ),
    ],
  );
}

/// 底部导航壳 + 全局离线提示条（数据来自兜底缓存时置顶警示）。
class HomeShell extends ConsumerStatefulWidget {
  const HomeShell({super.key, required this.child});

  final Widget child;

  @override
  ConsumerState<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends ConsumerState<HomeShell> {
  bool _offline = FallbackCache.I.offline;

  @override
  void initState() {
    super.initState();
    FallbackCache.I.onOfflineChanged((v) {
      if (mounted) setState(() => _offline = v);
    });
  }

  @override
  Widget build(BuildContext context) {
    final isEn = ref.watch(langProvider).code == 'en';
    final location = GoRouterState.of(context).matchedLocation;
    final index = _destinations
        .indexWhere((d) => location.startsWith(d.path))
        .clamp(0, _destinations.length - 1);
    return Scaffold(
      body: Column(
        children: [
          if (_offline)
            Material(
              color: const Color(0xFF3A2E12),
              child: SafeArea(
                bottom: false,
                child: Container(
                  width: double.infinity,
                  padding: const EdgeInsets.symmetric(vertical: 6),
                  alignment: Alignment.center,
                  child: Text(
                    isEn
                        ? '⚠ Network issue — showing cached data (may be stale)'
                        : '⚠ 网络异常，正在展示缓存数据，行情可能非实时',
                    style: const TextStyle(color: Color(0xFFF0B90B), fontSize: 12),
                  ),
                ),
              ),
            ),
          Expanded(child: widget.child),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: index,
        onDestinationSelected: (i) => context.go(_destinations[i].path),
        destinations: [
          for (final d in _destinations)
            NavigationDestination(
              icon: Icon(d.icon),
              selectedIcon: Icon(d.selectedIcon),
              label: isEn ? d.en : d.zh,
            ),
        ],
      ),
    );
  }
}
