import useSWR from 'swr'
import { useRouter } from 'next/router'

export default function Login({ children }) {
  const { data: auth, error } = useSWR('/api/users/self')
  const router = useRouter()
  const { next } = router.query

  if (!auth && !error) {
    return null
  }

  if (auth?.id) {
    if (next) {
      router.replace(decodeURIComponent(next))
    } else if (router.pathname !== '/login/finish') {
      // TODO (https://github.com/infrahq/infra/issues/1441): remove me when
      // using an OTP doesn't trigger authentication
      router.replace('/')
    }

    return false
  }

  return (
    <div className='flex min-h-full w-full flex-col justify-center'>
      <div className='mx-auto my-8 flex w-full max-w-xs flex-col items-center justify-center rounded-lg border border-gray-800 px-5 pt-8 pb-6'>
        <div className='mb-4 rounded-full border border-violet-200/25 p-2.5'>
          <img alt='infra icon' className='h-12 w-12' src='/infra-color.svg' />
        </div>
        {children}
      </div>
    </div>
  )
}
