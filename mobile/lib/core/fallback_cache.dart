import 'package:hive_flutter/hive_flutter.dart';

/// GET 响应的应用层缓存兜底：
///   成功 → 写 Hive（持久化，重启仍可用）
///   连接失败 → 命中缓存则返回旧数据，并置全局离线标记（UI 顶栏提示）
class FallbackCache {
  FallbackCache._();
  static final FallbackCache I = FallbackCache._();

  static const _boxName = 'fallback';
  Box _box = Hive.box('klines'); // 占位，open() 后替换

  Future<void> open() async {
    _box = await Hive.openBox(_boxName);
  }

  bool _offline = false;
  bool get offline => _offline;
  final _listeners = <void Function(bool)>[];

  void setOffline(bool v) {
    if (_offline == v) return;
    _offline = v;
    for (final l in _listeners) {
      l(v);
    }
  }

  void onOfflineChanged(void Function(bool) listener) {
    _listeners.add(listener);
  }

  void put(String key, dynamic data) {
    try {
      _box.put(key, {'data': data, 'ts': DateTime.now().millisecondsSinceEpoch});
    } catch (_) {}
  }

  dynamic get(String key) {
    final rec = _box.get(key) as Map?;
    return rec?['data'];
  }

  int ts(String key) {
    final rec = _box.get(key) as Map?;
    return (rec?['ts'] ?? 0) as int;
  }
}
