import { CustomSpin } from '@components/ui/CustomSpin';

interface LoadingOverlayProps {
  open?: boolean;
  tip?: string;
}

export function LoadingOverlay({ open = false, tip }: LoadingOverlayProps) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/30 backdrop-blur-[1px]">
      <div className="rounded-xl bg-white px-6 py-5 shadow-lg">
        <CustomSpin size="large" tip={tip} />
      </div>
    </div>
  );
}
