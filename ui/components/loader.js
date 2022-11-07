export default function Loader({ size = 10, fullscreen = false }) {
  return (
    <div
      className={`flex ${
        fullscreen ? 'my-32 h-full w-full' : 'min-h-[100px] py-4'
      } items-center justify-center`}
    >
      <svg
        xmlns='http://www.w3.org/2000/svg'
        viewBox='0 0 100 100'
        preserveAspectRatio='xMidYMid'
        className={`h-${size} w-${size} animate-spin-fast stroke-current text-gray-400`}
      >
        <circle
          cx='50'
          cy='50'
          fill='none'
          strokeWidth='1.5'
          r='24'
          strokeDasharray='113.09733552923255 39.69911184307752'
        ></circle>
      </svg>
    </div>
  )
}
