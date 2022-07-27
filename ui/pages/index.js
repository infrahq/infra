import { useRouter } from 'next/router'

export default function Index() {
  useRouter().replace('/destinations')
  return false
}
