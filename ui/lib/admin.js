import useSWR from 'swr'

import { useGrants } from './grants'

export function useAdmin () {
  const { data: auth } = useSWR('/api/users/self')
  const { grants } = useGrants({ resource: 'infra', user: auth?.id })

  const admin = !!grants?.find(g => g.privilege === 'admin')

  return {
    loading: !auth || !grants,
    admin
  }
}
