import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// 轻量双语（zh/en）。shared_preferences 持久化。
class LangState {
  const LangState({this.code = 'zh'});
  final String code;

  String t(String zh, String en) => code == 'en' ? en : zh;
}

class LangNotifier extends StateNotifier<LangState> {
  LangNotifier() : super(const LangState());

  Future<void> load() async {
    final sp = await SharedPreferences.getInstance();
    state = LangState(code: sp.getString('lang') ?? 'zh');
  }

  Future<void> toggle() async {
    final next = state.code == 'zh' ? 'en' : 'zh';
    state = LangState(code: next);
    final sp = await SharedPreferences.getInstance();
    await sp.setString('lang', next);
  }
}
