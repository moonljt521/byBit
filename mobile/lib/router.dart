import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'pages/assets_page.dart';
import 'pages/futures_page.dart';
import 'pages/learn_page.dart';
import 'pages/login_page.dart';
import 'pages/markets_page.dart';
import 'pages/mine_page.dart';
import 'pages/spot_page.dart';
import 'providers/app_providers.dart';

final rootNavigatorKey = GlobalKey<NavigatorState>();

/// 底部 Tab 路由 + 登录守卫。
GoRouter buildRouter(WidgetRef ref) {
  final auth = ref.watch(authProvider);
  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation: auth.loggedIn ? '/markets' : '/login',
    redirect: (context, state) {
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

/// 底部导航壳。
class HomeShell extends ConsumerWidget {
  const HomeShell({super.key, required this.child});

  final Widget child;

  static const _destinations = [
    ('/markets', '行情', 'Markets', Icons.candlestick_chart_outlined, Icons.candlestick_chart),
    ('/spot', '现货', 'Spot', Icons.swap_horiz_outlined, Icons.swap_horiz),
    ('/futures', '合约', 'Futures', Icons.show_chart_outlined, Icons.show_chart),
    ('/assets', '资产', 'Assets', Icons.account_balance_wallet_outlined, Icons.account_balance_wallet),
    ('/learn', '学习', 'Learn', Icons.menu_book_outlined, Icons.menu_book),
    ('/mine', '我的', 'Me', Icons.person_outline, Icons.person),
  ];

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isEn = ref.watch(langProvider).code == 'en';
    final location = GoRouterState.of(context).matchedLocation;
    final index =
        _destinations.indexWhere((d) => location.startsWith(d.$1)).clamp(0, _destinations.length - 1);
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: index,
        onDestinationSelected: (i) => context.go(_destinations[i].$1),
        destinations: [
          for (final d in _destinations)
            NavigationDestination(
              icon: Icon(d.$4),
              selectedIcon: Icon(d.$5),
              label: isEn ? d.$3 : d.$2,
            ),
        ],
      ),
    );
  }
}
