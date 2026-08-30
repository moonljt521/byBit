import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/api_client.dart';
import '../core/models.dart';
import '../core/theme.dart';
import '../core/widgets.dart';
import '../providers/app_providers.dart';

const futSymbols = ['BTCUSDT', 'ETHUSDT'];

/// 合约交易页：K 线 + 开仓（多空/杠杆/逐仓）+ 持仓与强平价 + 资金费率记录。
class FuturesPage extends ConsumerStatefulWidget {
  const FuturesPage({super.key});

  @override
  ConsumerState<FuturesPage> createState() => _FuturesPageState();
}

class _FuturesPageState extends ConsumerState<FuturesPage> {
  String _symbol = 'BTCUSDT';
  String _side = 'long';
  double _leverage = 5;
  final _amount = TextEditingController();
  List<Candle> _candles = [];
  Ticker? _ticker;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final results = await Future.wait([
        ApiClient.I.klines(_symbol, '1m'),
        ApiClient.I.tickers(),
      ]);
      if (!mounted) return;
      final tickers = results[1] as List<Ticker>;
      setState(() {
        _candles = results[0] as List<Candle>;
        for (final t in tickers) {
          if (t.symbol == _symbol) _ticker = t;
        }
      });
      ref.invalidate(positionsProvider);
    } catch (_) {}
  }

  Future<void> _open() async {
    final amount = _amount.text.trim();
    if (amount.isEmpty || (double.tryParse(amount) ?? 0) <= 0) {
      _snack('请输入有效数量', error: true);
      return;
    }
    setState(() => _busy = true);
    try {
      final pos = await ApiClient.I.openPosition(
        symbol: _symbol,
        side: _side,
        leverage: _leverage.round(),
        amount: amount,
      );
      if (!mounted) return;
      _snack('开${_side == 'long' ? '多' : '空'}成功，开仓价 ${pos.entryPrice}，强平价 ${pos.liquidationPrice}');
      _amount.clear();
      ref.invalidate(positionsProvider);
      ref.invalidate(balancesProvider);
    } on ApiException catch (e) {
      _snack(e.message, error: true);
    } catch (e) {
      _snack('网络异常：$e', error: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _close(Position p) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('平仓'),
        content: Text('按当前标记价 ${p.markPrice} 全部平仓并结算盈亏，确定？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(backgroundColor: AppTheme.down),
            child: const Text('平仓'),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ApiClient.I.closePosition(p.id);
      if (mounted) _snack('已平仓 #${p.id}');
      ref.invalidate(positionsProvider);
      ref.invalidate(balancesProvider);
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
    final positions = ref.watch(positionsProvider);
    final notional = (double.tryParse(_ticker?.last ?? '0') ?? 0) * (double.tryParse(_amount.text) ?? 0);
    return Scaffold(
      appBar: AppBar(
        title: DropdownButton<String>(
          value: _symbol,
          underline: const SizedBox(),
          items: futSymbols
              .map((s) => DropdownMenuItem(
                  value: s, child: Text('${s.replaceAll('USDT', '')}USDT 永续')))
              .toList(),
          onChanged: (v) => setState(() {
            _symbol = v ?? _symbol;
            _load();
          }),
        ),
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: CandleChart(candles: _candles, height: 200),
            ),
            const Divider(height: 24),
            Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(value: 'long', label: Text('开多（看涨）')),
                      ButtonSegment(value: 'short', label: Text('开空（看跌）')),
                    ],
                    selected: {_side},
                    onSelectionChanged: (s) => setState(() => _side = s.first),
                    style: ButtonStyle(
                      backgroundColor: WidgetStateProperty.resolveWith((states) =>
                          states.contains(WidgetState.selected)
                              ? (_side == 'long' ? AppTheme.up : AppTheme.down)
                              : null),
                      foregroundColor: WidgetStateProperty.resolveWith((states) =>
                          states.contains(WidgetState.selected) ? Colors.white : null),
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text('杠杆 ${_leverage.round()}x（强平距离约 ${(_leverage > 0 ? 100 / _leverage : 0).toStringAsFixed(1)}%）',
                      style: const TextStyle(fontSize: 13)),
                  Slider(
                    min: 1,
                    max: 20,
                    divisions: 19,
                    value: _leverage,
                    label: '${_leverage.round()}x',
                    onChanged: (v) => setState(() => _leverage = v),
                  ),
                  TextField(
                    controller: _amount,
                    keyboardType: TextInputType.number,
                    decoration: InputDecoration(
                        hintText:
                            '数量（${_symbol.replaceAll('USDT', '')}，最小名义价值 5 USDT）'),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    '名义价值 ${notional.toStringAsFixed(2)} USDT · 保证金 ${(notional / _leverage).toStringAsFixed(2)} · taker 费率 0.05%',
                    style: TextStyle(fontSize: 12, color: Colors.grey.shade500),
                  ),
                  const SizedBox(height: 10),
                  FilledButton(
                    onPressed: _busy ? null : _open,
                    style: FilledButton.styleFrom(
                      backgroundColor: _side == 'long' ? AppTheme.up : AppTheme.down,
                      minimumSize: const Size.fromHeight(48),
                    ),
                    child: Text(_busy
                        ? '提交中…'
                        : '开${_side == 'long' ? '多' : '空'} ${_symbol.replaceAll('USDT', '')} 永续'),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    '逐仓模式 · 维持保证金率 0.5% · 资金费率每 8 小时结算 0.01%（多头付空头）· 触及强平价损失全部保证金',
                    style: TextStyle(fontSize: 11, color: Colors.grey.shade600),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.all(12),
              child: Text('当前仓位', style: Theme.of(context).textTheme.titleMedium),
            ),
            positions.when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (e, _) => Padding(
                padding: const EdgeInsets.all(12),
                child: Text('加载失败：$e', style: const TextStyle(color: AppTheme.down)),
              ),
              data: (list) => list.isEmpty
                  ? const Padding(
                      padding: EdgeInsets.all(16),
                      child: Center(
                          child: Text('暂无仓位', style: TextStyle(color: Colors.grey))),
                    )
                  : Column(
                      children: list.map((p) {
                        final pnl = double.tryParse(p.unrealizedPnl) ?? 0;
                        return ListTile(
                          dense: true,
                          title: Text(
                            '${p.symbol.replaceAll('USDT', '/USDT')} ${p.leverage}x ${p.long ? '多' : '空'}',
                            style: TextStyle(
                                color: p.long ? AppTheme.up : AppTheme.down,
                                fontWeight: FontWeight.w600),
                          ),
                          subtitle: Text(
                              '数量 ${p.size} · 开仓 ${p.entryPrice} · 标记 ${p.markPrice}\n强平价 ${p.liquidationPrice} · 保证金 ${p.margin}'),
                          isThreeLine: true,
                          trailing: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            crossAxisAlignment: CrossAxisAlignment.end,
                            children: [
                              Text('浮盈 ${p.unrealizedPnl}',
                                  style: TextStyle(
                                      color: pnl >= 0 ? AppTheme.up : AppTheme.down,
                                      fontWeight: FontWeight.bold)),
                              Text('ROI ${p.roi}%',
                                  style: TextStyle(
                                      fontSize: 12, color: Colors.grey.shade500)),
                              TextButton(
                                onPressed: () => _close(p),
                                child: const Text('平仓',
                                    style: TextStyle(color: AppTheme.down)),
                              ),
                            ],
                          ),
                        );
                      }).toList(),
                    ),
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }
}
