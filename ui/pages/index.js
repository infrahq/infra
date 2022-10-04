import { useRouter } from 'next/router'

import { useUser } from '../lib/hooks'

export default function Index() {
  const { loading } = useUser({ redirectTo: '/login' })
  const router = useRouter()

  if (loading) {
    return null
  }

  if (router.isReady) {
    router.replace('/destinations')
  }

  return null
}
