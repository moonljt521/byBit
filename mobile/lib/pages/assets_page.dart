import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/api_client.dart';
import '../core/theme.dart';
import '../providers/app_providers.dart';

/// 资产页：余额 + 流水 + 一键重置。
class AssetsPage extends ConsumerWidget {
  const AssetsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final balances = ref.watch(balancesProvider);
    return Scaffold(
      appBar: AppBar(
        title: const Text('资产（虚拟）'),
        actions: [
          IconButton(
            tooltip: '重置账户',
            onPressed: () async {
              final ok = await showDialog<bool>(
                context: context,
                builder: (ctx) => AlertDialog(
                  title: const Text('重置账户'),
                  content: const Text('将清空全部虚拟持仓与余额，恢复为初始 10,000 USDT（历史流水保留）。确定？'),
                  actions: [
                    TextButton(
                        onPressed: () => Navigator.pop(ctx, false),
                        child: const Text('取消')),
                    FilledButton(
                      onPressed: () => Navigator.pop(ctx, true),
                      style: FilledButton.styleFrom(backgroundColor: AppTheme.down),
                      child: const Text('确定重置'),
                    ),
                  ],
                ),
              );
              if (ok != true) return;
              try {
                await ApiClient.I.resetAccount();
                ref.invalidate(balancesProvider);
              } on ApiException catch (e) {
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(e.message), backgroundColor: AppTheme.down));
                }
              }
            },
            icon: const Icon(Icons.restart_alt),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(balancesProvider),
        child: balances.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => Center(child: Text('加载失败：$e')),
          data: (list) => list.isEmpty
              ? ListView(children: const [
                  SizedBox(height: 120),
                  Center(child: Text('暂无资产', style: TextStyle(color: Colors.grey))),
                ])
              : ListView.separated(
                  itemCount: list.length,
                  separatorBuilder: (_, __) =>
                      const Divider(height: 1, indent: 16),
                  itemBuilder: (context, i) {
                    final b = list[i];
                    return ListTile(
                      leading: CircleAvatar(
                        backgroundColor: AppTheme.accent.withOpacity(0.15),
                        child: Text(
                          b.currency.substring(0, b.currency.length > 2 ? 2 : 1),
                          style: const TextStyle(
                              color: AppTheme.accent,
                              fontWeight: FontWeight.bold,
                              fontSize: 12),
                        ),
                      ),
                      title: Text(b.currency),
                      subtitle: Text('冻结 ${b.frozen}'),
                      trailing: Text(b.available,
                          style: const TextStyle(
                              fontWeight: FontWeight.bold, fontSize: 16)),
                    );
                  },
                ),
        ),
      ),
    );
  }
}
