import { useRouter } from 'next/router'
import { useEffect } from 'react'

import { useUser } from '../lib/hooks'

export default function Index() {
  const { loading } = useUser({ redirectTo: '/login' })
  const router = useRouter()

  useEffect(() => {
    router.replace('/destinations')
  }, [router])

  if (loading) {
    return null
  }

  return null
}
