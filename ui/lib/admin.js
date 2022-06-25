import useSWR from 'swr'

const INFRA_ADMIN_ROLE = 'admin'

export function useAdmin () {
  const { data: auth } = useSWR('/api/users/self', { revalidateIfStale: false })
  const { data: { items: grants } = {} } = useSWR(`/api/grants?user=${auth?.id}`, { revalidateIfStale: false })
  const { data: { items: groups } = {} } = useSWR(`/api/groups?userID=${auth?.id}`, { revalidateIfStale: false })

  // todo: switch to using /api/grants?inherited=1 instead of
  // multiple fetches (/api/grants for each group)
  const { data: groupGrantDatas } = useSWR(
    () => groups.map(g => `/api/grants?group=${g.id}`) || null,
    (...urls) => Promise.all(urls.map(url => fetch(url).then(r => r.json()))),
    { revalidateIfStale: false }
  )

  const inherited = groupGrantDatas?.map(g => g.items)?.flat()

  const merged = [...grants || [], ...inherited || []]
  const loading = [auth, grants, groups, groups?.length ? inherited : true].some(x => !x)
  const admin = merged.some(g => g.privilege === INFRA_ADMIN_ROLE)

  return {
    loading,
    admin
  }
}
