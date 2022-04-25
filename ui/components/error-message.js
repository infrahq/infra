export default function ({ message, center = false }) {
  return (
    <p className={`${center ? 'mt-2 text-center' : 'px-4 mb-1'} text-sm text-pink-500`}>{message}</p>
  )
}
