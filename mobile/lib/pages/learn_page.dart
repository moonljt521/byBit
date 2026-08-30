import 'package:flutter/material.dart';

import '../core/api_client.dart';
import '../core/models.dart';

/// 学习中心：币种百科 / 新手教程 / 术语词典。
class LearnPage extends StatelessWidget {
  const LearnPage({super.key});

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 3,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('学习中心'),
          bottom: const TabBar(tabs: [
            Tab(text: '币种百科'),
            Tab(text: '新手教程'),
            Tab(text: '术语词典'),
          ]),
        ),
        body: const TabBarView(
          children: [_DocList(kind: 'coins'), _DocList(kind: 'concepts'), _GlossaryTab()],
        ),
      ),
    );
  }
}

class _DocList extends StatefulWidget {
  const _DocList({required this.kind});

  final String kind;

  @override
  State<_DocList> createState() => _DocListState();
}

class _DocListState extends State<_DocList> with AutomaticKeepAliveClientMixin {
  List<LearnDoc> _items = [];

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final items = widget.kind == 'coins'
          ? await ApiClient.I.coins()
          : await ApiClient.I.concepts();
      if (mounted) setState(() => _items = items);
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    if (_items.isEmpty) {
      return const Center(child: Text('加载中…', style: TextStyle(color: Colors.grey)));
    }
    return ListView.builder(
      itemCount: _items.length,
      itemBuilder: (context, i) => ListTile(
        title: Text(_items[i].title),
        trailing: const Icon(Icons.chevron_right, color: Colors.grey),
        onTap: () => Navigator.push(
            context,
            MaterialPageRoute(
                builder: (_) =>
                    _ReaderPage(doc: _items[i], kind: widget.kind))),
      ),
    );
  }
}

class _ReaderPage extends StatefulWidget {
  const _ReaderPage({required this.doc, required this.kind});

  final LearnDoc doc;
  final String kind; // coins / concepts

  @override
  State<_ReaderPage> createState() => _ReaderPageState();
}

class _ReaderPageState extends State<_ReaderPage> {
  String _content = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final doc = widget.doc.content.isNotEmpty
          ? widget.doc
          : await ApiClient.I.doc(widget.kind, widget.doc.slug);
      if (mounted) setState(() => _content = doc.content);
    } catch (e) {
      if (mounted) setState(() => _content = '加载失败：$e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(widget.doc.title)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(_content.isEmpty ? '加载中…' : _content,
              style: const TextStyle(height: 1.7, fontSize: 15)),
        ],
      ),
    );
  }
}

class _GlossaryTab extends StatefulWidget {
  const _GlossaryTab();

  @override
  State<_GlossaryTab> createState() => _GlossaryTabState();
}

class _GlossaryTabState extends State<_GlossaryTab> {
  List<GlossaryTerm> _terms = [];
  String _kw = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final terms = await ApiClient.I.glossary();
      if (mounted) setState(() => _terms = terms);
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final filtered = _terms
        .where((t) =>
            _kw.isEmpty ||
            t.term.contains(_kw) ||
            t.en.toLowerCase().contains(_kw.toLowerCase()) ||
            t.definition.contains(_kw))
        .toList();
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: TextField(
            decoration: const InputDecoration(
              prefixIcon: Icon(Icons.search),
              hintText: '搜索术语，如：强平 / funding / 私钥',
            ),
            onChanged: (v) => setState(() => _kw = v.trim()),
          ),
        ),
        Expanded(
          child: ListView.separated(
            itemCount: filtered.length,
            separatorBuilder: (_, __) => const Divider(height: 1, indent: 16),
            itemBuilder: (context, i) {
              final t = filtered[i];
              return ListTile(
                title: Text('${t.term}  ',
                    style: const TextStyle(fontWeight: FontWeight.w600)),
                subtitle: Text(t.definition, style: const TextStyle(fontSize: 13)),
                trailing: Text(t.en,
                    style: TextStyle(fontSize: 12, color: Colors.grey.shade500)),
              );
            },
          ),
        ),
      ],
    );
  }
}
