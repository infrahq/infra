import { useRouter } from 'next/router'

import LoginLayout from '../../components/layouts/login'
import PasswordResetForm from '../../components/password-reset-form'

export default function AcceptInvite() {
  const router = useRouter()
  const { token } = router.query

  return (
    <>
      {token ? (
        <>
          <h2 className='my-3 max-w-[260px] text-center text-xs text-gray-300'>
            Welcome to InfraHQ! Please set your password
          </h2>
          <div className='relative mt-4 w-full'>
            <div
              className='absolute inset-0 flex items-center'
              aria-hidden='true'
            >
              <div className='w-full border-t border-gray-800' />
            </div>
          </div>
          <PasswordResetForm />
        </>
      ) : (
        <h1 className='text-base font-bold leading-snug'>Token missing</h1>
      )}
    </>
  )
}

AcceptInvite.layout = page => <LoginLayout>{page}</LoginLayout>
