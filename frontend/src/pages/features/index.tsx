import { DocsFeaturePhases, DocsListFeatures, DocsReadPhase } from '@wailsjs/go/main/App';
import { FileTextOutlined, ReloadOutlined } from '@ant-design/icons';
import { Button, Empty, Layout, Menu, Skeleton, Tabs, Typography } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { MarkdownPreview } from '@components/ui/MarkdownPreview';

const { Sider, Content } = Layout;
const { Title } = Typography;

function formatName(name: string) {
  return name
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function PhaseContent({ feature, phase }: { feature: string; phase: string }) {
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setContent(null);
    DocsReadPhase(feature, phase)
      .then(setContent)
      .finally(() => setLoading(false));
  }, [feature, phase]);

  if (loading) return <Skeleton active paragraph={{ rows: 10 }} />;

  return <MarkdownPreview content={content ?? ''} />;
}

export default function FeaturesPage() {
  const [features, setFeatures] = useState<string[]>([]);
  const [selectedFeature, setSelectedFeature] = useState<string | null>(null);
  const [phases, setPhases] = useState<string[]>([]);
  const [activePhase, setActivePhase] = useState<string | null>(null);
  const [loadingFeatures, setLoadingFeatures] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  const loadFeatures = useCallback(() => {
    setLoadingFeatures(true);
    DocsListFeatures()
      .then((list) => {
        setFeatures(list ?? []);
        setSelectedFeature((prev) => {
          if (prev && list?.includes(prev)) return prev;
          return list?.[0] ?? null;
        });
      })
      .finally(() => setLoadingFeatures(false));
  }, []);

  useEffect(() => {
    loadFeatures();
  }, [loadFeatures, refreshKey]);

  useEffect(() => {
    if (!selectedFeature) return;
    setPhases([]);
    setActivePhase(null);
    DocsFeaturePhases(selectedFeature).then((list) => {
      setPhases(list ?? []);
      if (list?.length) setActivePhase(list[0]);
    });
  }, [selectedFeature, refreshKey]);

  const featureMenuItems = features.map((f) => ({
    key: f,
    icon: <FileTextOutlined />,
    label: formatName(f),
  }));

  const phaseTabs = phases.map((p) => ({ key: p, label: formatName(p) }));

  return (
    <Layout
      className="rounded-lg overflow-hidden"
      style={{ background: 'transparent', height: 'calc(100vh - 112px)' }}
    >
      <Sider
        width={220}
        style={{ background: 'var(--ant-color-bg-container)', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        className="rounded-l-lg"
      >
        <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between flex-shrink-0">
          <Title level={5} className="!mb-0">
            Features
          </Title>
          <Button
            type="text"
            size="small"
            icon={<ReloadOutlined />}
            loading={loadingFeatures}
            onClick={() => setRefreshKey((k) => k + 1)}
          />
        </div>
        <div className="overflow-y-auto flex-1">
          {loadingFeatures ? (
            <div className="p-4">
              <Skeleton active paragraph={{ rows: 5 }} title={false} />
            </div>
          ) : (
            <Menu
              mode="inline"
              selectedKeys={selectedFeature ? [selectedFeature] : []}
              items={featureMenuItems}
              onSelect={({ key }) => setSelectedFeature(key)}
              style={{ border: 'none' }}
            />
          )}
        </div>
      </Sider>

      <Content
        style={{ background: 'var(--ant-color-bg-layout)', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
      >
        {!selectedFeature ? (
          <div className="flex items-center justify-center h-full">
            <Empty description="Select a feature from the sidebar" />
          </div>
        ) : phases.length === 0 ? (
          <div className="p-6"><Skeleton active paragraph={{ rows: 8 }} /></div>
        ) : (
          <>
            {/* Fixed: feature title + phase tabs bar */}
            <div className="px-6 pt-6 flex-shrink-0">
              <Title level={4} className="!mb-2">
                {formatName(selectedFeature)}
              </Title>
              <Tabs
                activeKey={activePhase ?? undefined}
                items={phaseTabs}
                onChange={setActivePhase}
              />
            </div>
            {/* Scrollable: markdown content only */}
            <div className="flex-1 overflow-y-auto px-6 pb-6">
              {activePhase && (
                <PhaseContent
                  key={`${selectedFeature}-${activePhase}`}
                  feature={selectedFeature}
                  phase={activePhase}
                />
              )}
            </div>
          </>
        )}
      </Content>
    </Layout>
  );
}
