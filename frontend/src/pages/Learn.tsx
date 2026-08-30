import { useEffect, useMemo, useState } from 'react'
import { Card, Col, Empty, Input, Menu, Row, Table, Tabs, Typography } from 'antd'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { getCoin, getConcept, glossary, listCoins, listConcepts } from '../api/learn'
import type { GlossaryTerm, LearnDoc, LearnItem } from '../api/learn'

function Reader({ doc }: { doc: LearnDoc | null }) {
  if (!doc) return <Empty description="选择左侧文章阅读" style={{ marginTop: 80 }} />;
  // 正文自带的一级标题与页面标题重复，去掉首个 H1
  const body = doc.content.replace(/^#\s+.+\n/, '');
  return (
    <div style={{ maxWidth: 860, lineHeight: 1.9 }}>
      <Typography.Title level={3}>{doc.title}</Typography.Title>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          table: (props) => (
            <table style={{ borderCollapse: 'collapse' }} {...props}>
              {props.children}
            </table>
          ),
          th: (props) => (
            <th
              style={{ border: '1px solid #ddd', padding: '6px 12px', background: '#fafafa' }}
              {...props}
            />
          ),
          td: (props) => <td style={{ border: '1px solid #ddd', padding: '6px 12px' }} {...props} />,
          a: (props) => <a target="_blank" rel="noreferrer" {...props} />,
          code: (props) => (
            <code style={{ background: '#f5f5f5', padding: '2px 6px', borderRadius: 4 }} {...props} />
          ),
        }}
      >
        {body}
      </ReactMarkdown>
    </div>
  )
}

function DocExplorer(props: {
  items: LearnItem[]
  load: (slug: string) => Promise<LearnDoc>
  defaultSlug?: string
}) {
  const [selected, setSelected] = useState(props.defaultSlug ?? props.items[0]?.slug ?? '')
  const [doc, setDoc] = useState<LearnDoc | null>(null)
  useEffect(() => {
    if (!selected) return
    void props
      .load(selected)
      .then(setDoc)
      .catch(() => {})
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected])
  return (
    <Row gutter={16}>
      <Col span={7}>
        <Menu
          mode="inline"
          selectedKeys={[selected]}
          onClick={({ key }) => setSelected(key)}
          items={props.items.map((i) => ({ key: i.slug, label: i.title }))}
          style={{ maxHeight: 620, overflowY: 'auto', border: '1px solid #f0f0f0', borderRadius: 8 }}
        />
      </Col>
      <Col span={17}>
        <Reader doc={doc} />
      </Col>
    </Row>
  )
}

function GlossaryTab() {
  const [terms, setTerms] = useState<GlossaryTerm[]>([])
  const [kw, setKw] = useState('')
  useEffect(() => {
    void glossary().then(setTerms).catch(() => {})
  }, [])
  const filtered = useMemo(
    () =>
      terms.filter(
        (t) =>
          !kw ||
          t.term.includes(kw) ||
          t.en.toLowerCase().includes(kw.toLowerCase()) ||
          t.definition.includes(kw),
      ),
    [terms, kw],
  )
  return (
    <div>
      <Input.Search
        placeholder="搜索术语，如：强平 / funding / 私钥"
        style={{ width: 360, marginBottom: 12 }}
        onSearch={setKw}
        onChange={(e) => !e.target.value && setKw('')}
        allowClear
      />
      <Table
        rowKey="term"
        size="small"
        dataSource={filtered}
        pagination={{ pageSize: 20 }}
        columns={[
          { title: '术语', dataIndex: 'term', width: 120, render: (v: string) => <b>{v}</b> },
          { title: '英文', dataIndex: 'en', width: 200, render: (v: string) => <Typography.Text type="secondary">{v}</Typography.Text> },
          { title: '解释', dataIndex: 'definition' },
        ]}
      />
    </div>
  )
}

export default function Learn() {
  const [coins, setCoins] = useState<LearnItem[]>([])
  const [concepts, setConcepts] = useState<LearnItem[]>([])
  useEffect(() => {
    void listCoins().then(setCoins).catch(() => {})
    void listConcepts().then(setConcepts).catch(() => {})
  }, [])
  return (
    <Card size="small">
      <Tabs
        items={[
          {
            key: 'coins',
            label: '币种百科',
            children: coins.length ? (
              <DocExplorer items={coins} load={getCoin} defaultSlug="btc" />
            ) : (
              <Empty description="加载中" />
            ),
          },
          {
            key: 'concepts',
            label: '新手教程',
            children: concepts.length ? (
              <DocExplorer items={concepts} load={getConcept} defaultSlug="blockchain" />
            ) : (
              <Empty description="加载中" />
            ),
          },
          { key: 'glossary', label: '术语词典', children: <GlossaryTab /> },
        ]}
      />
    </Card>
  )
}
