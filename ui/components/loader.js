export default function Loader(props) {
  return (
    <svg
      {...props}
      xmlns='http://www.w3.org/2000/svg'
      viewBox='0 0 100 100'
      preserveAspectRatio='xMidYMid'
      className={`${props.className} animate-spin-fast stroke-current text-gray-400`}
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
  )
}
