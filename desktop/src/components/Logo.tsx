export function Logo({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      fill="none"
      aria-hidden="true"
      className={className}
    >
      <rect width="32" height="32" rx="9" fill="#10141a" />
      <path
        d="M11 8v16"
        stroke="#5eead4"
        strokeWidth="2.4"
        strokeLinecap="round"
      />
      <path
        d="M11 16c5 0 5-6 10-6"
        stroke="#5eead4"
        strokeWidth="2.4"
        strokeLinecap="round"
      />
      <circle cx="11" cy="8" r="2.2" fill="#5eead4" />
      <circle cx="21" cy="10" r="2.2" fill="#5eead4" />
      <circle cx="11" cy="24" r="2.2" fill="#5eead4" />
    </svg>
  );
}
