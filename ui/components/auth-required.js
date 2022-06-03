import { useRouter } from 'next/router'
import useSWR from 'swr'

export default function ({ children }) {
  const { data: auth, error } = useSWR('/api/users/self')
  const router = useRouter()

  if (!auth && !error) {
    return null
  }

  if (!auth?.id) {
    router.replace('/login')
    return null
  }

  return children
}
