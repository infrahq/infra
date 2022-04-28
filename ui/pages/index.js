import { useRouter } from 'next/router'

export default function () {
  useRouter().replace('/destinations')

  return null
}
