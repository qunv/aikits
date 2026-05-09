import { FolderOpenOutlined, FolderOutlined } from '@ant-design/icons';
import { SelectRepository } from '@wailsjs/go/main/App';
import { Button, Tooltip, Typography } from 'antd';
import { observer } from 'mobx-react-lite';
import { useState } from 'react';
import { useRepoStore } from '@stores/StoreContext';

export const RepoSelector = observer(() => {
  const repoStore = useRepoStore();
  const [loading, setLoading] = useState(false);

  const handleSelect = async () => {
    setLoading(true);
    try {
      const path = await SelectRepository();
      if (path) repoStore.setRepoPath(path);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Tooltip title={repoStore.repoPath || 'Select a repository'} placement="bottom">
      <Button
        type="text"
        loading={loading}
        icon={repoStore.repoPath ? <FolderOpenOutlined /> : <FolderOutlined />}
        onClick={handleSelect}
        className="flex items-center gap-1 max-w-[220px]"
      >
        {repoStore.repoName ? (
          <Typography.Text ellipsis className="max-w-[160px] text-inherit">
            {repoStore.repoName}
          </Typography.Text>
        ) : (
          <span className="text-sm">Open repository…</span>
        )}
      </Button>
    </Tooltip>
  );
});
