import { useRouter } from 'next/router'
import { useEffect } from 'react'

export default function Index() {
  const router = useRouter()

  useEffect(() => {
    if (!router.isReady) return
    // wait for router to be ready to prevent router from being rendered server-side
    router.replace('/destinations')
  })

  return null
}
