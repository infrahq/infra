import useSWR from 'swr'

export function useServerConfig() {
  const { data: { isEmailConfigured, isSignupEnabled } = {} } = useSWR(
    `/api/configuration`,
    {
      revalidateIfStale: false,
    }
  )

  return {
    isEmailConfigured,
    isSignupEnabled,
  }
}
