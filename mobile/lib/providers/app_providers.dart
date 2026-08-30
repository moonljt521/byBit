import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../core/lang.dart';

import '../core/api_client.dart';
import '../core/models.dart';

/// 认证状态（登录/登出后通知路由守卫刷新）。
class AuthState {
  const AuthState({required this.loggedIn, required this.username});
  final bool loggedIn;
  final String username;
}

class AuthNotifier extends StateNotifier<AuthState> {
  // main() 已在 runApp 前加载完凭证，构造即持有登录态 → 冷启动不闪登录页
  AuthNotifier()
      : super(AuthState(
            loggedIn: CredentialStore.I.loggedIn,
            username: CredentialStore.I.username));

  void refresh() {
    final cred = CredentialStore.I;
    state = AuthState(loggedIn: cred.loggedIn, username: cred.username);
  }

  Future<void> logout() async {
    await CredentialStore.I.clearSession();
    state = const AuthState(loggedIn: false, username: '');
  }
}

final authProvider =
    StateNotifierProvider<AuthNotifier, AuthState>((ref) => AuthNotifier());

/// 行情列表（自动轮询由 UI 层 refetch 触发）。
final tickersProvider =
    FutureProvider.autoDispose<List<Ticker>>((ref) => ApiClient.I.tickers());

/// 现货当前委托。
final openOrdersProvider =
    FutureProvider.autoDispose<List<Order>>((ref) => ApiClient.I.openOrders());

/// 合约持仓。
final positionsProvider =
    FutureProvider.autoDispose<List<Position>>((ref) => ApiClient.I.positions());

/// 资产余额。
final balancesProvider =
    FutureProvider.autoDispose<List<Balance>>((ref) => ApiClient.I.balances());

/// 语言状态（zh/en）。
final langProvider = StateNotifierProvider<LangNotifier, LangState>((ref) => LangNotifier());
