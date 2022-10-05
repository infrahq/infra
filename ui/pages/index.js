import { useRouter } from 'next/router'
import { useEffect } from 'react'

import Dashboard from '../components/layouts/dashboard'

export default function Index() {
  const router = useRouter()

  useEffect(() => {
    router.replace('/destinations')
  }, [router])

  return null
}

Index.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
