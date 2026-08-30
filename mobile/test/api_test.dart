import 'package:flutter_test/flutter_test.dart';

import 'package:cryptosim_mobile/core/api_client.dart';
import 'package:cryptosim_mobile/core/models.dart';

void main() {
  test('HMAC 签名算法符合后端规范', () {
    // 与后端 middleware.Private 相同输入应产生确定性签名（此处验证稳定性与格式）
    final sig1 = signRequest(
      secret: 'secret',
      ts: '1700000000',
      method: 'GET',
      fullPath: '/api/v1/account/me',
      query: '',
      body: '',
    );
    final sig2 = signRequest(
      secret: 'secret',
      ts: '1700000000',
      method: 'GET',
      fullPath: '/api/v1/account/me',
      query: '',
      body: '',
    );
    expect(sig1, sig2);
    expect(sig1.length, 64); // hex(sha256)
    // 不同输入应产生不同签名
    final sig3 = signRequest(
      secret: 'secret',
      ts: '1700000001',
      method: 'GET',
      fullPath: '/api/v1/account/me',
      query: '',
      body: '',
    );
    expect(sig1, isNot(sig3));
  });

  test('Ticker 模型解析', () {
    final t = Ticker.fromJson({
      'symbol': 'BTCUSDT',
      'last': '78000.1',
      'change_pct': '-1.20',
      'high24h': '79000',
      'low24h': '77000',
      'vol24h': '12345',
    });
    expect(t.symbol, 'BTCUSDT');
    expect(t.up, isFalse);
  });

  test('Candle 模型解析与涨跌判断', () {
    final c = Candle.fromJson({
      'ts': 1725000000000, 'o': '100', 'h': '110', 'l': '95', 'c': '105', 'vol': '1.5',
    });
    expect(c.up, isTrue);
    expect(c.close, 105);
  });

  test('Position 模型解析', () {
    final p = Position.fromJson({
      'id': 1, 'symbol': 'BTCUSDT', 'side': 'long', 'leverage': 5,
      'size': '0.01', 'entry_price': '78000', 'mark_price': '78100',
      'unrealized_pnl': '0.2', 'liquidation_price': '62700', 'roi': '2.56',
    });
    expect(p.long, isTrue);
    expect(p.leverage, 5);
  });
}
