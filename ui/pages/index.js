import { useRouter } from 'next/router'

export default function Index() {
  const router = useRouter()

  if (!router.isReady) {
    return null
  }

  router.replace('/destinations')

  return null
}
