import 'dart:async';
import 'dart:convert';

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
    this.token = token;
    this.apiKey = apiKey;
    this.apiSecret = apiSecret;
    this.username = username;
    await _storage.write(key: 'token', value: token);
    await _storage.write(key: 'apiKey', value: apiKey);
    await _storage.write(key: 'apiSecret', value: apiSecret);
    await _storage.write(key: 'username', value: username);
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

/// 统一 API 客户端：dio + 鉴权/验签拦截器 + 统一业务错误。
class ApiClient {
  ApiClient._() {
    _dio = Dio(BaseOptions(
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 12),
    ));
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: _onRequest,
      onError: _onError,
    ));
    FallbackCache.I.open();
  }

  static bool _failoverInProgress = false;

  /// 连接类错误（IP 变了/后端不可达）→ 自动扫描局域网找新地址 → 重发原请求。
  Future<void> _onError(
    DioException e,
    ErrorInterceptorHandler handler,
  ) async {
    final isConnectionIssue = e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout;
    final alreadyRetried = e.requestOptions.extra['failoverRetried'] == true;

    if (isConnectionIssue && !_failoverInProgress && !alreadyRetried) {
      _failoverInProgress = true;
      try {
        final found = await ServerDiscovery.I.discover();
        if (found != null) {
          await CredentialStore.I.setBaseUrl(found);
          final opts = e.requestOptions..extra['failoverRetried'] = true;
          try {
            final resp = await _dio.fetch(opts); // 重发（重新签名）
            _failoverInProgress = false;
            handler.resolve(resp);
            return;
          } on DioException catch (e2) {
            _failoverInProgress = false;
            return handler.next(e2);
          }
        }
      } catch (_) {
        // 发现失败，按原错误返回
      }
      _failoverInProgress = false;
    }
    handler.next(e);
  }

  static final ApiClient I = ApiClient._();
  late final Dio _dio;

  Future<void> _onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final cred = CredentialStore.I;

    // 自动故障转移重试：path 已是绝对 URL（上次失败的请求），仅替换 host 后直通。
    // HMAC 签名只覆盖 path+query+body，与 host 无关，原签名可直接复用。
    final asUri = Uri.tryParse(options.path);
    if (asUri != null && asUri.hasScheme && asUri.host.isNotEmpty) {
      final newBase = Uri.parse(cred.baseUrl);
      final rewritten = newBase.replace(
        path: asUri.path,
        query: asUri.query.isEmpty ? null : asUri.query,
      );
      options
        ..queryParameters = const {}
        ..path = rewritten.toString();
      handler.next(options);
      return;
    }

    final base = cred.baseUrl;
    final signPath = Uri.parse(base).path + options.path; // 含 /api/v1 前缀，用于签名

    // query 统一按 key 字典序编码，保证与签名一致，并直接拼入 URL
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

  static String _cacheKey(String path, Map<String, dynamic>? query) {
    final qs = query == null
        ? ''
        : (query.keys.toList()..sort()).map((k) => '$k=${query[k]}').join('&');
    return '$path?$qs';
  }

  /// GET：成功写缓存；连接失败（后端挂了/断网）→ 回退缓存 + 离线标记。
  Future<dynamic> get(String path, [Map<String, dynamic>? query]) async {
    final key = _cacheKey(path, query);
    try {
      final data = _unwrap(await _dio.get(path, queryParameters: query));
      FallbackCache.I.setOffline(false);
      unawaited(FallbackCache.I.put(key, data)); // 异步写缓存，不阻塞响应
      return data;
    } on DioException catch (e) {
      if (e.type == DioExceptionType.connectionError ||
          e.type == DioExceptionType.connectionTimeout) {
        final cached = await FallbackCache.I.get(key);
        if (cached != null) {
          FallbackCache.I.setOffline(true);
          return cached; // 旧数据兜底（已 JSON 归一化，调用方无感知）
        }
        FallbackCache.I.setOffline(true);
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

  ApiException _fromDio(DioException e) {
    final resp = e.response;
    if (resp?.data is Map<String, dynamic>) {
      final body = resp!.data as Map<String, dynamic>;
      final code = (body['code'] ?? -1) as int;
      if (code == 10401) CredentialStore.I.clearSession();
      return ApiException(code, (body['msg'] ?? '请求失败') as String);
    }
    // 连接类错误给用户可懂的文案，而不是 dio 原始报错
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout) {
      return ApiException(-1, '网络连接失败：后端服务未启动或网络不可用，已展示最近的缓存数据');
    }
    return ApiException(-1, '网络异常：${e.message}');
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

  /// K 线：优先 Hive 缓存（离线可见），成功后写缓存。
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
      rethrow;
    } catch (_) {
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
