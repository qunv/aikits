import logo from '@/assets/images/logo-universal.png';

interface LogoProps {
  invert?: boolean;
  collapsed?: boolean;
}

export function Logo({ invert, collapsed }: LogoProps) {
  return (
    <div className="relative overflow-hidden transition-all duration-200">
      <img
        src={logo}
        alt="Aikits Logo"
        width={collapsed ? 32 : 100}
        className={invert ? '[filter:invert(0.5)_brightness(2)]' : undefined}
      />
    </div>
  );
}
