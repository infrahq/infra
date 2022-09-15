import { useRouter } from 'next/router'

import LoginLayout from '../../components/layouts/login'
import PasswordResetForm from '../../components/password-reset-form'

export default function AcceptInvite() {
  const router = useRouter()
  const { token } = router.query

  if (!router.isReady) {
    return null
  }

  if (!token) {
    router.replace('/')
    return null
  }

  return (
    <div className='flex min-h-[320px] w-full flex-col items-center px-10 py-8'>
      <h1 className='text-base font-bold leading-snug'>Welcome to Infra</h1>
      <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-500'>
        Please set your password to continue
      </h2>
      <PasswordResetForm />
    </div>
  )
}

AcceptInvite.layout = page => <LoginLayout>{page}</LoginLayout>
