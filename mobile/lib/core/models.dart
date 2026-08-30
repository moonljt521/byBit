/// 数据模型（与后端 JSON 一一对应）。
library;

class User {
  User.fromJson(Map<String, dynamic> j)
      : id = j['id'] as int,
        email = j['email'] as String? ?? '',
        username = j['username'] as String,
        role = j['role'] as String? ?? 'user';

  final int id;
  final String email;
  final String username;
  final String role;
}

class Ticker {
  Ticker.fromJson(Map<String, dynamic> j)
      : symbol = j['symbol'] as String,
        last = j['last'] as String,
        changePct = j['change_pct'] as String,
        high = j['high24h'] as String,
        low = j['low24h'] as String,
        vol = j['vol24h'] as String;

  final String symbol;
  final String last;
  final String changePct;
  final String high;
  final String low;
  final String vol;

  bool get up => (double.tryParse(changePct) ?? 0) >= 0;
}

class Candle {
  Candle.fromJson(Map<String, dynamic> j)
      : ts = j['ts'] as int,
        open = double.parse(j['o'] as String),
        high = double.parse(j['h'] as String),
        low = double.parse(j['l'] as String),
        close = double.parse(j['c'] as String),
        volume = double.parse(j['vol'] as String);

  final int ts;
  final double open;
  final double high;
  final double low;
  final double close;
  final double volume;

  bool get up => close >= open;

  Map<String, dynamic> toJson() => {
        'ts': ts, 'o': '$open', 'h': '$high', 'l': '$low', 'c': '$close', 'vol': '$volume',
      };
}

class DepthLevel {
  DepthLevel(this.price, this.size);
  final String price;
  final String size;
}

class Depth {
  Depth.fromJson(Map<String, dynamic> j)
      : bids = _lv(j['bids']),
        asks = _lv(j['asks']);

  static List<DepthLevel> _lv(dynamic rows) => (rows as List<dynamic>)
      .map((e) => DepthLevel((e as List<dynamic>)[0] as String, (e)[1] as String))
      .toList();

  final List<DepthLevel> bids;
  final List<DepthLevel> asks;
}

class Balance {
  Balance.fromJson(Map<String, dynamic> j)
      : currency = j['currency'] as String,
        available = j['available'] as String,
        frozen = j['frozen'] as String;

  final String currency;
  final String available;
  final String frozen;
}

class Order {
  Order.fromJson(Map<String, dynamic> j)
      : id = j['id'] as int,
        symbol = j['symbol'] as String,
        side = j['side'] as String,
        type = j['type'] as String,
        price = j['price'] as String,
        amount = j['amount'] as String,
        filled = j['filled'] as String,
        avgPrice = j['avg_price'] as String,
        fee = j['fee'] as String,
        status = j['status'] as String,
        createdAt = j['created_at'] as String;

  final int id;
  final String symbol;
  final String side;
  final String type;
  final String price;
  final String amount;
  final String filled;
  final String avgPrice;
  final String fee;
  final String status;
  final String createdAt;

  bool get open => status == 'pending' || status == 'partial';
}

class Position {
  Position.fromJson(Map<String, dynamic> j)
      : id = j['id'] as int,
        symbol = j['symbol'] as String,
        side = j['side'] as String,
        leverage = j['leverage'] as int,
        size = j['size'] as String,
        entryPrice = j['entry_price'] as String,
        margin = j['margin'] as String? ?? '0',
        markPrice = j['mark_price'] as String,
        unrealizedPnl = j['unrealized_pnl'] as String,
        liquidationPrice = j['liquidation_price'] as String,
        roi = j['roi'] as String;

  final int id;
  final String symbol;
  final String side;
  final int leverage;
  final String size;
  final String entryPrice;
  final String margin;
  final String markPrice;
  final String unrealizedPnl;
  final String liquidationPrice;
  final String roi;

  bool get long => side == 'long';
}

class FundingRecord {
  FundingRecord.fromJson(Map<String, dynamic> j)
      : id = j['id'] as int,
        symbol = j['symbol'] as String,
        rate = j['rate'] as String,
        amount = j['amount'] as String,
        createdAt = j['created_at'] as String;

  final int id;
  final String symbol;
  final String rate;
  final String amount;
  final String createdAt;
}

class LedgerEntry {
  LedgerEntry.fromJson(Map<String, dynamic> j)
      : id = j['id'] as int,
        bizType = j['biz_type'] as String,
        currency = j['currency'] as String,
        amount = j['amount'] as String,
        balanceAfter = j['balance_after'] as String,
        memo = j['memo'] as String,
        createdAt = j['created_at'] as String;

  final int id;
  final String bizType;
  final String currency;
  final String amount;
  final String balanceAfter;
  final String memo;
  final String createdAt;
}

class LearnDoc {
  LearnDoc.fromJson(Map<String, dynamic> j)
      : slug = j['slug'] as String,
        title = j['title'] as String,
        content = j['content'] as String? ?? '';

  final String slug;
  final String title;
  final String content;
}

class GlossaryTerm {
  GlossaryTerm.fromJson(Map<String, dynamic> j)
      : term = j['term'] as String,
        en = j['en'] as String,
        definition = j['definition'] as String;

  final String term;
  final String en;
  final String definition;
}
