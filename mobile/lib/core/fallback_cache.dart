import 'dart:async';
import 'dart:convert';

import 'package:hive_flutter/hive_flutter.dart';

/// GET 响应的应用层缓存兜底：
///   成功 → 写 Hive（持久化，重启仍可用）
///   连接失败 → 命中缓存则返回旧数据，并置全局离线标记（UI 顶栏提示）
class FallbackCache {
  FallbackCache._();
  static final FallbackCache I = FallbackCache._();

  static const _boxName = 'fallback';
  final Completer<Box> _ready = Completer<Box>();

  Future<void> open() async {
    final box = await Hive.openBox(_boxName);
    if (!_ready.isCompleted) _ready.complete(box);
  }

  Future<Box> _box() async {
    if (!_ready.isCompleted) await open();
    return _ready.future;
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

  /// 写入前做 JSON 归一化：保证读出的是 Map<String, dynamic>/List，调用方可直接 cast。
  Future<void> put(String key, dynamic data) async {
    try {
      final box = await _box();
      final normalized = data is String ? data : jsonDecode(jsonEncode(data));
      await box.put(key, {'data': normalized, 'ts': DateTime.now().millisecondsSinceEpoch});
    } catch (_) {
      // 缓存失败不影响主流程
    }
  }

  Future<dynamic> get(String key) async {
    try {
      final box = await _box();
      final rec = box.get(key) as Map?;
      final data = rec?['data'];
      if (data == null) return null;
      // Hive 反序列化不保留 Map 泛型：同一会话内写入的还能读出
      // Map<String, dynamic>，但跨进程从磁盘读回的全是 Map<dynamic, dynamic>，
      // 调用方 `as Map<String, dynamic>` 会直接抛类型异常。这里统一深归一化。
      return _normalize(data);
    } catch (_) {
      return null;
    }
  }

  /// 深度归一化：所有层级的 Map 转成 Map<String, dynamic>，List 逐项递归。
  static dynamic _normalize(dynamic v) {
    if (v is Map) {
      return v.map((k, e) => MapEntry(k.toString(), _normalize(e)));
    }
    if (v is List) {
      return [for (final e in v) _normalize(e)];
    }
    return v;
  }

  Future<int> ts(String key) async {
    final box = await _box();
    final rec = box.get(key) as Map?;
    return (rec?['ts'] ?? 0) as int;
  }
}
