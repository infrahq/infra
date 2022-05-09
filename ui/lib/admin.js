import useSWR from 'swr'

export function useAdmin () {
  const { data: auth } = useSWR('/v1/identities/self')
  const { data: grants, error: grantsError } = useSWR(() => `/v1/identities/${auth?.id}/grants?resource=infra`)

  return {
    loading: !grants && !grantsError,
    admin: !!grants?.items?.find(g => g.privilege === 'admin')
  }
}
