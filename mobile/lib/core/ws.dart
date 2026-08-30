import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import 'api_client.dart';
import 'discovery.dart';
import 'models.dart';

/// 实时行情流：WebSocket 订阅（断线自动重连），无连接时回退轮询由 UI 处理。
class MarketStream {
  MarketStream._();
  static final MarketStream I = MarketStream._();

  final _controller = StreamController<List<Ticker>>.broadcast();
  WebSocketChannel? _channel;

  int _retryCount = 0;

  Stream<List<Ticker>> subscribe() {
    _connect();
    return _controller.stream;
  }

  void _connect() {
    final cred = CredentialStore.I;
    final wsBase = cred.baseUrl.replaceFirst(RegExp(r'^http'), 'ws');
    final token = cred.token;
    _channel = WebSocketChannel.connect(Uri.parse(
      '$wsBase/ws/market?channels=tickers${token.isNotEmpty ? '&token=$token' : ''}',
    ));
    _channel!.stream.listen(
      (data) {
        try {
          final msg = jsonDecode(data as String) as Map<String, dynamic>;
          if (msg['channel'] == 'tickers') {
            final list = (msg['data'] as List<dynamic>)
                .map((e) => Ticker.fromJson(e as Map<String, dynamic>))
                .toList();
            _controller.add(list);
          }
        } catch (_) {}
      },
      onDone: () async {
        _retryCount = (_retryCount + 1).clamp(0, 5);
        // 重连前自动发现：IP 可能变了
        if (await ServerDiscovery.I.discover() != null) _retryCount = 0;
        Timer(Duration(seconds: 2 * _retryCount), _connect);
      },
      onError: (_) {},
    );
  }
}
