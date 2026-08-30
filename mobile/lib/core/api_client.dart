import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:hive_flutter/hive_flutter.dart';

import 'discovery.dart';
import 'fallback_cache.dart';
import 'models.dart';

class ApiException implements Exception {
  ApiException(this.code, this.message);
  final int code;
  final String message;
  @override
  String toString() => message;
}

/// HMAC 请求签名（与后端 middleware.Private 规范一致）：
/// stringToSign = ts \n METHOD \n fullPath \n sortedQuery \n sha256hex(body)
String signRequest({
  required String secret,
  required String ts,
  required String method,
  required String fullPath,
  required String query,
  required String body,
}) {
  final bodyHash = sha256.convert(utf8.encode(body)).toString();
  final sts = '$ts\n${method.toUpperCase()}\n$fullPath\n$query\n$bodyHash';
  return Hmac(sha256, utf8.encode(secret)).convert(utf8.encode(sts)).toString();
}

/// 凭证持久化：flutter_secure_storage（Android Keystore / iOS Keychain 加密）。
class CredentialStore {
  CredentialStore._();
  static final CredentialStore I = CredentialStore._();

  static const _storage = FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );

  String token = '';
  String apiKey = '';
  String apiSecret = '';
  String username = '';
  String baseUrl = 'http://10.0.2.2:8080/api/v1';

  bool get loggedIn => token.isNotEmpty && apiKey.isNotEmpty;

  Future<void> load() async {
    token = await _storage.read(key: 'token') ?? '';
    apiKey = await _storage.read(key: 'apiKey') ?? '';
    apiSecret = await _storage.read(key: 'apiSecret') ?? '';
    username = await _storage.read(key: 'username') ?? '';
    baseUrl = await _storage.read(key: 'baseUrl') ?? baseUrl;
  }

  Future<void> saveSession({
    required String token,
    required String apiKey,
    required String apiSecret,
    required String username,
  }) async {
    const sp = _storage;
    await sp.write(key: 'token', value: token);
    await sp.write(key: 'apiKey', value: apiKey);
    await sp.write(key: 'apiSecret', value: apiSecret);
    await sp.write(key: 'username', value: username);
  }

  Future<void> setBaseUrl(String v) async {
    baseUrl = v.endsWith('/') ? v.substring(0, v.length - 1) : v;
    await _storage.write(key: 'baseUrl', value: baseUrl);
  }

  Future<void> clearSession() async {
    token = apiKey = apiSecret = username = '';
    for (final k in ['token', 'apiKey', 'apiSecret', 'username']) {
      await _storage.delete(key: k);
    }
  }
}

/// 统一 API 客户端：缓存优先（stale-while-revalidate）+ 自动故障转移。
class ApiClient {
  ApiClient._() {
    _dio = Dio(BaseOptions(
      connectTimeout: const Duration(seconds: 4), // 局域网足够；快速失败以尽快回退缓存
      receiveTimeout: const Duration(seconds: 12),
    ));
    _dio.interceptors.add(InterceptorsWrapper(onRequest: _onRequest));
  }

  static final ApiClient I = ApiClient._();
  late final Dio _dio;
  Timer? _probe;

  Future<void> _onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final cred = CredentialStore.I;
    final base = cred.baseUrl;
    final signPath = Uri.parse(base).path + options.path; // 含 /api/v1 前缀，用于签名

    final qp = options.queryParameters;
    final sortedKeys = qp.keys.toList()..sort();
    final query = sortedKeys
        .map((k) => '$k=${Uri.encodeComponent(qp[k].toString())}')
        .join('&');

    final body = options.data != null ? jsonEncode(options.data) : '';
    if (cred.loggedIn) {
      final ts = (DateTime.now().millisecondsSinceEpoch ~/ 1000).toString();
      options.headers['X-API-KEY'] = cred.apiKey;
      options.headers['X-API-TIMESTAMP'] = ts;
      options.headers['X-API-SIGNATURE'] = signRequest(
        secret: cred.apiSecret,
        ts: ts,
        method: options.method,
        fullPath: signPath,
        query: query,
        body: body,
      );
      options.headers['Authorization'] = 'Bearer ${cred.token}';
    }
    options
      ..queryParameters = const {}
      ..path = '$base${options.path}${query.isEmpty ? '' : '?$query'}'; // 绝对 URL
    handler.next(options);
  }

  dynamic _unwrap(Response resp) {
    final body = resp.data;
    if (body is! Map<String, dynamic>) throw ApiException(-1, '响应格式异常');
    final code = (body['code'] ?? -1) as int;
    if (code == 0) return body['data'];
    throw ApiException(code, (body['msg'] ?? '请求失败') as String);
  }

  bool _isConnectionError(DioException e) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout) {
      return true;
    }
    return e.type == DioExceptionType.unknown && e.error is SocketException;
  }

  ApiException _fromDio(DioException e) {
    final resp = e.response;
    if (resp?.data is Map<String, dynamic>) {
      final body = resp!.data as Map<String, dynamic>;
      final code = (body['code'] ?? -1) as int;
      if (code == 10401) CredentialStore.I.clearSession();
      return ApiException(code, (body['msg'] ?? '请求失败') as String);
    }
    if (_isConnectionError(e)) {
      return ApiException(-1, '网络连接失败：后端服务未启动或网络不可用，已展示最近的缓存数据');
    }
    return ApiException(-1, '网络异常：${e.message}');
  }

  static String _cacheKey(String path, Map<String, dynamic>? query) {
    final qs = query == null
        ? ''
        : (query.keys.toList()..sort()).map((k) => '$k=${query[k]}').join('&');
    return '$path?$qs';
  }

  /// 同步读取某接口的缓存快照（供页面 init 时立即渲染，随后再网络刷新）。
  Future<dynamic> peek(String path, [Map<String, dynamic>? query]) =>
      FallbackCache.I.get(_cacheKey(path, query));

  Future<dynamic> get(String path, [Map<String, dynamic>? query]) async {
    final key = _cacheKey(path, query);
    try {
      final data = _unwrap(await _dio.get(path, queryParameters: query));
      FallbackCache.I.setOffline(false);
      _stopProbe();
      unawaited(FallbackCache.I.put(key, data));
      return data;
    } on DioException catch (e) {
      if (!_isConnectionError(e)) {
        throw _fromDio(e);
      }
      final cached = await FallbackCache.I.get(key);
      if (cached != null) {
        FallbackCache.I.setOffline(true);
        _startProbe();
        return cached; // 缓存先行，恢复探测放后台
      }
      // 无缓存：尝试自动故障转移（找到新地址并重发一次）
      FallbackCache.I.setOffline(true);
      _startProbe();
      final opts = await _tryFailover(e.requestOptions);
      if (opts != null) {
        final data = _unwrap(await _dio.fetch(opts));
        unawaited(FallbackCache.I.put(key, data));
        return data;
      }
      throw _fromDio(e);
    } on ApiException {
      FallbackCache.I.setOffline(false);
      rethrow;
    }
  }

  Future<dynamic> post(String path, [Map<String, dynamic>? body]) async {
    try {
      final data = _unwrap(await _dio.post(path, data: body ?? {}));
      FallbackCache.I.setOffline(false);
      _stopProbe();
      return data;
    } on DioException catch (e) {
      throw _fromDio(e);
    }
  }

  Future<dynamic> delete(String path) async {
    try {
      return _unwrap(await _dio.delete(path));
    } on DioException catch (e) {
      throw _fromDio(e);
    }
  }

  /// 后台恢复：离线期间周期性探测后端；成功即解除离线标记。
  void _startProbe() {
    if (_probe != null) return;
    _probe = Timer.periodic(const Duration(seconds: 15), (_) async {
      try {
        final resp = await _dio.get<void>('/health',
            options: Options(sendTimeout: const Duration(seconds: 2), receiveTimeout: const Duration(seconds: 2)));
        if (resp.statusCode == 200) {
          _stopProbe(); // 下一次正常请求会清除离线标记
        }
      } catch (_) {}
    });
  }

  void _stopProbe() {
    _probe?.cancel();
    _probe = null;
  }

  /// 局域网自动发现新后端地址并重写请求。找不到返回 null。
  Future<RequestOptions?> _tryFailover(RequestOptions original) async {
    try {
      final found = await ServerDiscovery.I.discover();
      if (found == null) return null;
      await CredentialStore.I.setBaseUrl(found);
      final asUri = Uri.parse(original.path);
      final newBase = Uri.parse(found);
      final rewritten = newBase.replace(
        path: asUri.path,
        query: asUri.query.isEmpty ? null : asUri.query,
      );
      return original..path = rewritten.toString();
    } catch (_) {
      return null;
    }
  }

  // ---------- 业务接口 ----------

  Future<void> register(String email, String username, String password) async {
    final data = await post('/auth/register', {
      'email': email, 'username': username, 'password': password,
    }) as Map<String, dynamic>;
    await CredentialStore.I.saveSession(
      token: data['token'] as String,
      apiKey: data['api_key'] as String,
      apiSecret: data['api_secret'] as String,
      username: username,
    );
  }

  Future<void> login(String account, String password) async {
    final data = await post('/auth/login', {
      'account': account, 'password': password,
    }) as Map<String, dynamic>;
    final user = data['user'] as Map<String, dynamic>;
    await CredentialStore.I.saveSession(
      token: data['token'] as String,
      apiKey: data['api_key'] as String,
      apiSecret: data['api_secret'] as String,
      username: user['username'] as String,
    );
  }

  Future<void> resetCredentials() async {
    final data = await post('/auth/credentials/reset') as Map<String, dynamic>;
    final cred = CredentialStore.I;
    await cred.saveSession(
      token: cred.token,
      apiKey: data['api_key'] as String,
      apiSecret: data['api_secret'] as String,
      username: cred.username,
    );
  }

  Future<List<Ticker>> tickers() async {
    final data = await get('/market/tickers') as Map<String, dynamic>;
    return (data['tickers'] as List<dynamic>)
        .map((e) => Ticker.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<List<Ticker>> cachedTickers() async {
    final data = await peek('/market/tickers') as Map<String, dynamic>?;
    if (data == null) return const [];
    return (data['tickers'] as List<dynamic>)
        .map((e) => Ticker.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  /// K 线：网络失败时回退 Hive 持久缓存（本函数自己的兜底盒）。
  Future<List<Candle>> klines(String symbol, String bar, {int limit = 200}) async {
    final box = await Hive.openBox('klines');
    final cacheKey = '$symbol|$bar';
    try {
      final data = await get('/market/klines', {
        'symbol': symbol, 'bar': bar, 'limit': '$limit',
      }) as Map<String, dynamic>;
      final list = (data['candles'] as List<dynamic>)
          .map((e) => Candle.fromJson(e as Map<String, dynamic>))
          .toList();
      await box.put(cacheKey, list.map((c) => c.toJson()).toList());
      return list;
    } on ApiException {
      // 网络失败 → 回退本函数自身的 K 线缓存
      final cached = box.get(cacheKey) as List<dynamic>?;
      if (cached == null) rethrow;
      return cached
          .map((e) => Candle.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    }
  }

  Future<Depth> depth(String symbol, {int size = 10}) async {
    final data = await get('/market/depth', {'symbol': symbol, 'size': '$size'})
        as Map<String, dynamic>;
    return Depth.fromJson(data);
  }

  Future<List<Balance>> balances() async {
    final data = await get('/account/me') as Map<String, dynamic>;
    return (data['balances'] as List<dynamic>)
        .map((e) => Balance.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<void> resetAccount() => post('/account/reset');

  Future<Order?> placeOrder({
    required String symbol,
    required String side,
    required String type,
    String? price,
    required String amount,
    String? clientOrderId,
  }) async {
    final data = await post('/spot/orders', {
      'symbol': symbol, 'side': side, 'type': type,
      if (price != null && price.isNotEmpty) 'price': price,
      'amount': amount,
      if (clientOrderId != null) 'client_order_id': clientOrderId,
    });
    return data == null ? null : Order.fromJson(data as Map<String, dynamic>);
  }

  Future<List<Order>> openOrders() async {
    final list = await get('/spot/orders/open') as List<dynamic>;
    return list.map((e) => Order.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<void> cancelOrder(int id) => delete('/spot/orders/$id');

  Future<Position> openPosition({
    required String symbol,
    required String side,
    required int leverage,
    required String amount,
  }) async {
    final data = await post('/futures/positions', {
      'symbol': symbol, 'side': side, 'leverage': leverage, 'amount': amount,
    }) as Map<String, dynamic>;
    return Position.fromJson(data);
  }

  Future<void> closePosition(int id) => post('/futures/positions/$id/close');

  Future<List<Position>> positions() async {
    final list = await get('/futures/positions') as List<dynamic>;
    return list.map((e) => Position.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<List<FundingRecord>> fundingRecords() async {
    final list = await get('/futures/funding') as List<dynamic>;
    return list.map((e) => FundingRecord.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<List<LearnDoc>> coins() async {
    final list = await get('/learn/coins') as List<dynamic>;
    return list.map((e) => LearnDoc.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<LearnDoc> doc(String kind, String slug) async {
    final data = await get('/learn/$kind/$slug') as Map<String, dynamic>;
    return LearnDoc.fromJson(data);
  }

  Future<List<LearnDoc>> concepts() async {
    final list = await get('/learn/concepts') as List<dynamic>;
    return list.map((e) => LearnDoc.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<List<GlossaryTerm>> glossary() async {
    final list = await get('/learn/glossary') as List<dynamic>;
    return list.map((e) => GlossaryTerm.fromJson(e as Map<String, dynamic>)).toList();
  }
}
