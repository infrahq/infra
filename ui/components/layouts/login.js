import { useRouter } from 'next/router'
import { useUser } from '../../lib/hooks'

export default function Login({ children }) {
  const router = useRouter()
  const { next } = router.query

  const { loading, user } = useUser()

  if (loading) {
    return null
  }

  if (user) {
    router.replace(next ? decodeURIComponent(next) : '/')
    return null
  }

  return (
    <div className='flex min-h-screen flex-col justify-center py-20'>
      <div className='flex w-full flex-col items-center justify-center rounded-xl md:mx-auto md:max-w-sm'>
        {children}
      </div>

      {/* Infra logotype */}
      <svg
        viewBox='0 0 192 74'
        fill='none'
        xmlns='http://www.w3.org/2000/svg'
        className='mt-6 mb-16 h-4 fill-current text-black/[7%]'
      >
        <g clipPath='url(#clip0_170_6)'>
          <path d='M8.6 0C3.9 0 0 3.9 0 8.5C0 13.2 3.9 17.1 8.6 17.1C13.3 17.1 17.2 13.2 17.2 8.5C17.2 3.9 13.3 0 8.6 0ZM1.6 23.8V72.9H15.7V23.8H1.6Z' />
          <path d='M50.2211 22.8C44.7211 22.8 39.5211 25.4 36.8211 29.6V23.8H22.7211V72.9H36.8211V42.4C38.8211 39 42.7211 36.8 46.6211 36.8C52.4211 36.8 56.1211 40.7 56.1211 47.5V72.9H70.2211V45.1C70.2211 31.3 62.6211 22.8 50.2211 22.8Z' />
          <path d='M94.2898 20.9C94.2898 16.4 97.7898 13.4 101.89 13.4C103.29 13.4 105.29 13.8 106.29 14.1L106.99 1.7C105.19 0.800003 102.39 0.2 99.7898 0.2C88.2898 0.2 80.1898 7.5 80.1898 20.2V23.8H71.0898V37.2H80.1898V72.9H94.2898V37.2H106.39V23.8H94.2898V20.9Z' />
          <path d='M136.566 22.8C132.466 22.8 127.166 25.4 124.766 29.6V23.8H110.666V72.9H124.766V43.2C127.166 38.4 132.266 35.8 135.466 35.8C137.466 35.8 139.866 36.2 141.766 36.9L142.566 23.7C141.066 23.1 139.066 22.8 136.566 22.8Z' />
          <path d='M178.025 23.8V29.6C175.625 25.6 169.625 22.8 163.525 22.8C150.725 22.8 140.625 34.1 140.625 48.3C140.625 62.5 150.725 73.9 163.525 73.9C169.725 73.9 175.625 71.1 178.025 67.1V72.9H192.125V23.8H178.025ZM166.625 60.5C159.725 60.5 154.425 55.3 154.425 48.3C154.425 41.3 159.725 36.1 166.625 36.1C171.525 36.1 175.925 38.8 178.025 42.4V54.4C175.925 57.9 171.525 60.5 166.625 60.5Z' />
        </g>
      </svg>
    </div>
  )
}
