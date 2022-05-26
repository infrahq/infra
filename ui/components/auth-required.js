import { useRouter } from 'next/router'
import useSWR from 'swr'

export default function ({ children }) {
  const { data: auth, error } = useSWR('/api/users/self')
  const { data: signup } = useSWR('/api/signup')
  const router = useRouter()

  if (!signup) {
    return null
  }

  if (signup.enabled) {
    router.replace('/signup')
    return null
  }

  if (!auth && !error) {
    return null
  }

  if (!auth?.id) {
    router.replace('/login')
    return null
  }

  return children
}
