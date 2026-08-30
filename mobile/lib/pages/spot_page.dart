import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/api_client.dart';
import '../core/models.dart';
import '../core/theme.dart';
import '../core/widgets.dart';
import '../providers/app_providers.dart';

const symbols = ['BTCUSDT', 'ETHUSDT', 'BNBUSDT', 'SOLUSDT', 'XRPUSDT', 'TRXUSDT', 'DOGEUSDT'];
const bars = ['1m', '5m', '15m', '1h', '4h', '1d'];

/// 现货交易页：K 线 + 盘口 + 下单 + 当前委托。
class SpotPage extends ConsumerStatefulWidget {
  const SpotPage({super.key});

  @override
  ConsumerState<SpotPage> createState() => _SpotPageState();
}

class _SpotPageState extends ConsumerState<SpotPage> {
  String _symbol = 'BTCUSDT';
  String _bar = '1m';
  String _side = 'buy';
  String _type = 'limit';
  List<Candle> _candles = [];
  Ticker? _ticker;
  List<DepthLevel> _bids = [];
  List<DepthLevel> _asks = [];
  final _price = TextEditingController();
  final _amount = TextEditingController();
  Timer? _timer;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _loadCached(); // 缓存优先立即渲染
    _load();
    _timer = Timer.periodic(const Duration(seconds: 6), (_) => _load());
  }

  Future<void> _loadCached() async {
    final candles = await ApiClient.I.peek(
        '/market/klines', {'symbol': _symbol, 'bar': _bar, 'limit': '200'});
    if (mounted && candles is Map) {
      final list = (candles['candles'] as List)
          .map((e) => Candle.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
      setState(() => _candles = list);
    }
    final depth = await ApiClient.I.peek(
        '/market/depth', {'symbol': _symbol, 'size': '10'});
    if (mounted && depth is Map) {
      final d = Depth.fromJson(Map<String, dynamic>.from(depth));
      setState(() {
        _bids = d.bids.take(8).toList();
        _asks = d.asks.take(8).toList().reversed.toList();
      });
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    _price.dispose();
    _amount.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final results = await Future.wait([
        ApiClient.I.klines(_symbol, _bar),
        ApiClient.I.tickers(),
        ApiClient.I.depth(_symbol),
      ]);
      if (!mounted) return;
      final tickers = results[1] as List<Ticker>;
      final depth = results[2] as Depth;
      setState(() {
        _candles = results[0] as List<Candle>;
        for (final t in tickers) {
          if (t.symbol == _symbol) _ticker = t;
        }
        _bids = depth.bids.take(8).toList();
        _asks = depth.asks.take(8).toList().reversed.toList();
      });
    } catch (_) {
      // 静默重试（K 线有 Hive 缓存时离线也能显示）
    }
  }

  Future<void> _submit() async {
    final amount = _amount.text.trim();
    if (amount.isEmpty || (double.tryParse(amount) ?? 0) <= 0) {
      _snack('请输入有效数量', error: true);
      return;
    }
    setState(() => _busy = true);
    try {
      final order = await ApiClient.I.placeOrder(
        symbol: _symbol,
        side: _side,
        type: _type,
        price: _type == 'limit' ? _price.text.trim() : null,
        amount: amount,
        clientOrderId:
            'm-${DateTime.now().millisecondsSinceEpoch}', // 客户端幂等号
      );
      if (!mounted) return;
      _snack(_type == 'market'
          ? '市价单已成交，均价 ${order?.avgPrice ?? '-'}'
          : '限价单已提交，撮合引擎将自动撮合');
      _amount.clear();
      ref.invalidate(openOrdersProvider);
      ref.invalidate(balancesProvider);
    } on ApiException catch (e) {
      _snack(e.message, error: true);
    } catch (e) {
      _snack('网络异常：$e', error: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _cancel(Order o) async {
    try {
      await ApiClient.I.cancelOrder(o.id);
      if (mounted) _snack('已撤销 #${o.id}');
      ref.invalidate(openOrdersProvider);
    } on ApiException catch (e) {
      _snack(e.message, error: true);
    }
  }

  void _snack(String msg, {bool error = false}) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), backgroundColor: error ? AppTheme.down : null),
    );
  }

  @override
  Widget build(BuildContext context) {
    final orders = ref.watch(openOrdersProvider);
    final base = _symbol.replaceAll('USDT', '');
    return Scaffold(
      appBar: AppBar(
        title: DropdownButton<String>(
          value: _symbol,
          underline: const SizedBox(),
          items: symbols
              .map((s) => DropdownMenuItem(
                  value: s, child: Text('${s.replaceAll('USDT', '')}/USDT')))
              .toList(),
          onChanged: (v) => setState(() {
            _symbol = v ?? _symbol;
            _load();
          }),
        ),
        actions: [
          if (_ticker != null)
            Padding(
              padding: const EdgeInsets.only(right: 16),
              child: Center(
                child: Text(_ticker!.last,
                    style: TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.bold,
                        color: _ticker!.up ? AppTheme.up : AppTheme.down)),
              ),
            ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: CandleChart(candles: _candles),
            ),
            SizedBox(
              height: 44,
              child: ListView(
                scrollDirection: Axis.horizontal,
                children: bars
                    .map((b) => Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 4),
                          child: ChoiceChip(
                            label: Text(b.toUpperCase()),
                            selected: b == _bar,
                            onSelected: (_) => setState(() {
                              _bar = b;
                              _load();
                            }),
                          ),
                        ))
                    .toList(),
              ),
            ),
            const Divider(height: 1),
            // 盘口
            Padding(
              padding: const EdgeInsets.all(12),
              child: Row(
                children: [
                  Expanded(
                    child: _DepthColumn(
                        title: '买盘', levels: _bids.reversed.toList(), color: AppTheme.up),
                  ),
                  const SizedBox(width: 24),
                  Expanded(
                    child: _DepthColumn(title: '卖盘', levels: _asks, color: AppTheme.down),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            // 下单面板
            Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(value: 'buy', label: Text('买入')),
                      ButtonSegment(value: 'sell', label: Text('卖出')),
                    ],
                    selected: {_side},
                    onSelectionChanged: (s) => setState(() => _side = s.first),
                    style: ButtonStyle(
                      backgroundColor: WidgetStateProperty.resolveWith((states) =>
                          states.contains(WidgetState.selected)
                              ? (_side == 'buy' ? AppTheme.up : AppTheme.down)
                              : null),
                      foregroundColor: WidgetStateProperty.resolveWith((states) =>
                          states.contains(WidgetState.selected) ? Colors.white : null),
                    ),
                  ),
                  const SizedBox(height: 8),
                  SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(value: 'limit', label: Text('限价单')),
                      ButtonSegment(value: 'market', label: Text('市价单')),
                    ],
                    selected: {_type},
                    onSelectionChanged: (s) => setState(() => _type = s.first),
                  ),
                  const SizedBox(height: 8),
                  if (_type == 'limit')
                    TextField(
                      controller: _price,
                      keyboardType: TextInputType.number,
                      decoration: InputDecoration(
                          hintText: '价格（最新 ${_ticker?.last ?? '-'}）'),
                    ),
                  if (_type == 'limit') const SizedBox(height: 8),
                  TextField(
                    controller: _amount,
                    keyboardType: TextInputType.number,
                    decoration: InputDecoration(
                        hintText: '数量（$base，最小金额 5 USDT）'),
                  ),
                  const SizedBox(height: 10),
                  FilledButton(
                    onPressed: _busy ? null : _submit,
                    style: FilledButton.styleFrom(
                      backgroundColor: _side == 'buy' ? AppTheme.up : AppTheme.down,
                      minimumSize: const Size.fromHeight(48),
                    ),
                    child: Text('${_side == 'buy' ? '买入' : '卖出'} $base'),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            // 当前委托
            Padding(
              padding: const EdgeInsets.all(12),
              child: Text('当前委托',
                  style: Theme.of(context).textTheme.titleMedium),
            ),
            orders.when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (e, _) => Padding(
                padding: const EdgeInsets.all(12),
                child: Text('加载失败：$e', style: const TextStyle(color: AppTheme.down)),
              ),
              data: (list) => list.isEmpty
                  ? const Padding(
                      padding: EdgeInsets.all(16),
                      child: Center(
                          child: Text('暂无挂单', style: TextStyle(color: Colors.grey))),
                    )
                  : Column(
                      children: list
                          .map((o) => ListTile(
                                dense: true,
                                title: Text(
                                  '${o.side == 'buy' ? '买入' : '卖出'} ${o.symbol.replaceAll('USDT', '/USDT')}  @${o.price}',
                                  style: TextStyle(
                                      color: o.side == 'buy' ? AppTheme.up : AppTheme.down,
                                      fontWeight: FontWeight.w600),
                                ),
                                subtitle: Text(
                                    '数量 ${o.amount} · 已成交 ${o.filled} · ${_status(o.status)}'),
                                trailing: TextButton(
                                  onPressed: () => _cancel(o),
                                  child: const Text('撤销',
                                      style: TextStyle(color: AppTheme.down)),
                                ),
                              ))
                          .toList(),
                    ),
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  String _status(String s) => {
        'pending': '等待成交',
        'partial': '部分成交',
        'filled': '已成交',
        'canceled': '已撤销',
      }[s] ?? s;
}

class _DepthColumn extends StatelessWidget {
  const _DepthColumn({required this.title, required this.levels, required this.color});

  final String title;
  final List<DepthLevel> levels;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title, style: TextStyle(fontSize: 12, color: Colors.grey.shade500)),
        const SizedBox(height: 4),
        ...levels.map((l) => Padding(
              padding: const EdgeInsets.symmetric(vertical: 1),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(l.price, style: TextStyle(fontSize: 12, color: color)),
                  Text((double.tryParse(l.size) ?? 0).toStringAsFixed(4),
                      style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
                ],
              ),
            )),
      ],
    );
  }
}
