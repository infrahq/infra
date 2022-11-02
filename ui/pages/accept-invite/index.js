import { useRouter } from 'next/router'

import Login from '../../components/layouts/login'
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
    <div className='flex w-full flex-col items-center px-10 pt-4 pb-6'>
      <PasswordResetForm
        header='Welcome to Infra'
        subheader='Please set your password to continue'
      />
    </div>
  )
}

AcceptInvite.layout = page => <Login>{page}</Login>
