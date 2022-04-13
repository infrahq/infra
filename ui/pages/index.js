import { useRouter } from 'next/router'

export default function () {
  const router = useRouter()

  router.replace('/destinations')

  return null
}
