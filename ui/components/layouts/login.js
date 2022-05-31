import useSWR from 'swr'
import { useRouter } from 'next/router'

export default function ({ children }) {
  const { data: auth, error } = useSWR('/api/users/self')
  const router = useRouter()

  if (!auth && !error) {
    return null
  }

  if (auth?.id) {
    // TODO (https://github.com/infrahq/infra/issues/1441): remove me when
    // using an OTP doesn't trigger authentication
    if (router.pathname !== '/login/finish') {
      router.replace('/')
      return null
    }
  }

  return (
    <div className='w-full min-h-full flex flex-col justify-center'>
      <div className='flex flex-col w-full max-w-xs mx-auto justify-center items-center my-8 px-5 pt-8 pb-4 border rounded-lg border-gray-800'>
        <div className='border border-violet-200/25 rounded-full p-2.5 mb-4'>
          <img className='w-12 h-12' src='/infra-color.svg' />
        </div>
        {children}
      </div>
    </div>
  )
}
