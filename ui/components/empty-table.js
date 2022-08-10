export default function EmptyTable({
  title,
  subtitle,
  icon,
  buttonText,
  buttonHref,
}) {
  return (
    <div className='mx-auto my-20 flex flex-1 flex-col justify-center text-center'>
      <span className='mx-auto my-4 h-7 w-7'>{icon}</span>
      <h1 className='mb-2 text-base font-bold'>{title}</h1>
      <h2 className='mx-auto mb-4 max-w-xs text-2xs text-gray-400'>
        {subtitle}
      </h2>
    </div>
  )
}
