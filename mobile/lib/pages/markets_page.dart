import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../core/api_client.dart';
import '../core/models.dart';
import '../core/ws.dart';
import '../core/theme.dart';
import '../providers/app_providers.dart';

/// 行情页：实时币价（Riverpod 管理状态，下拉刷新 + 定时刷新）。
class MarketsPage extends ConsumerStatefulWidget {
  const MarketsPage({super.key});

  @override
  ConsumerState<MarketsPage> createState() => _MarketsPageState();
}

class _MarketsPageState extends ConsumerState<MarketsPage> {
  StreamSubscription<List<Ticker>>? _wsSub;
  List<Ticker>? _wsTickers;

  @override
  void initState() {
    super.initState();
    _schedule();
    // 缓存优先：立即渲染上次数据，再走网络刷新
    ApiClient.I.cachedTickers().then((t) {
      if (mounted && _wsTickers == null && t.isNotEmpty) {
        setState(() => _wsTickers = t);
      }
    });
    // WebSocket 实时推送（REST 30 秒轮询作兜底）
    _wsSub = MarketStream.I.subscribe().listen((t) {
      if (mounted) setState(() => _wsTickers = t);
    });
  }

  void _schedule() {
    Future.delayed(const Duration(seconds: 30), () {
      if (mounted) {
        ref.invalidate(tickersProvider);
        _schedule();
      }
    });
  }

  @override
  void dispose() {
    _wsSub?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final tickers = _wsTickers ?? ref.watch(tickersProvider).valueOrNull ?? const <Ticker>[];
    return Scaffold(
      appBar: AppBar(
        title: const Text('行情'),
        actions: [
          IconButton(
            onPressed: () => ref.invalidate(tickersProvider),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(tickersProvider),
        child: tickers.isEmpty
            ? ListView(children: [
                const SizedBox(height: 120),
                const Center(
                    child: Text('暂无行情数据：网络不可用且本地无缓存\n启动后端或恢复网络后点「重试」',
                        style: TextStyle(color: Colors.grey))),
                const SizedBox(height: 16),
                Center(
                  child: OutlinedButton(
                    onPressed: () => ref.invalidate(tickersProvider),
                    child: const Text('重试'),
                  ),
                ),
              ])
            : ListView.separated(
                itemCount: tickers.length,
                separatorBuilder: (_, __) =>
                    const Divider(height: 1, indent: 16),
                itemBuilder: (context, i) => _TickerTile(t: tickers[i]),
              ),
      ),
    );
  }
}

class _TickerTile extends StatelessWidget {
  const _TickerTile({required this.t});

  final Ticker t;

  @override
  Widget build(BuildContext context) {
    final color = t.up ? AppTheme.up : AppTheme.down;
    return ListTile(
      title: Text(
        '${t.symbol.replaceAll('USDT', '')}/USDT',
        style: const TextStyle(fontWeight: FontWeight.bold),
      ),
      subtitle: Text(
        '24h量 ${double.tryParse(t.vol)?.toStringAsFixed(0) ?? t.vol}',
        style: TextStyle(fontSize: 12, color: Colors.grey.shade500),
      ),
      trailing: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          Text(t.last, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
          Text('${t.up ? '+' : ''}${t.changePct}%',
              style: TextStyle(color: color, fontSize: 13)),
        ],
      ),
      onTap: () => context.go('/spot'),
    );
  }
}
