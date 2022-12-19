import { useRouter } from 'next/router'
import { useServerConfig } from '../../lib/serverconfig'

import Login from '../../components/layouts/login'
import Providers, { oidcLogin } from '../../components/providers'
import PasswordResetForm from '../../components/password-reset-form'

export default function AcceptInvite() {
  const router = useRouter()
  const { token } = router.query
  const { baseDomain, loginDomain, google } = useServerConfig()

  if (!router.isReady) {
    return null
  }

  if (!token) {
    router.replace('/')
    return null
  }

  return (
    <div className='flex w-full flex-col items-center px-10 pt-4 pb-6'>
      <h1 className='text-base font-bold leading-snug'>Welcome to Infra</h1>
      {google && (
        <>
          <Providers
            providers={[google]}
            authnFunc={oidcLogin}
            baseDomain={baseDomain}
            loginDomain={loginDomain}
            buttonPrompt={'Log in with'}
          />
          <div className='relative mt-6 mb-2 w-full'>
            <div
              className='absolute inset-0 flex items-center'
              aria-hidden='true'
            >
              <div className='w-full border-t border-gray-200' />
            </div>
            <div className='relative flex justify-center text-sm'>
              <span className='bg-white px-2 text-2xs text-gray-400'>OR</span>
            </div>
          </div>
        </>
      )}
      <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-500'>
        set a password to continue
      </h2>
      <PasswordResetForm
        header='Welcome to Infra'
        subheader='Please set your password to continue'
      />
    </div>
  )
}

AcceptInvite.layout = page => <Login>{page}</Login>
