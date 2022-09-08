import { useRouter } from 'next/router'
import useSWR, { useSWRConfig } from 'swr'
import { useEffect, useState } from 'react'
import Head from 'next/head'
import Link from 'next/link'

import { useAdmin } from '../../../lib/admin'
import { sortByPrivilege } from '../../../lib/grants'

import Breadcrumbs from '../../../components/breadcrumbs'
import AccessTable from '../../../components/access-table'
import GrantForm from '../../../components/grant-form'
import RemoveButton from '../../../components/remove-button'
import Dashboard from '../../../components/layouts/dashboard'

function parent(resource = '') {
  const parts = resource.split('.')
  return parts.length > 1 ? parts[0] : null
}

function ConnectSection({ roles, resource, kind = 'resource' }) {
  return (
    <div>
      <p className='my-4 text-sm leading-normal text-gray-500'>
        Connect to this {kind} via the{' '}
        <a
          target='_blank'
          href='https://infrahq.com/docs/install/install-infra-cli'
          className='font-medium text-blue-600 underline hover:text-blue-500'
          rel='noreferrer'
        >
          Infra CLI
        </a>
        . You have <span className='font-semibold'>{roles.join(', ')}</span>{' '}
        access.
      </p>
      <pre className='overflow-auto rounded-md bg-gray-900 px-4 py-3 text-2xs leading-normal text-gray-300'>
        infra login {window.location.host}
        <br />
        infra use {resource}
        <br />
        kubectl get pods
      </pre>
    </div>
  )
}

function NamespacesTable({ resources, destinationId }) {
  return (
    <table className='min-w-full divide-y divide-gray-300'>
      <tbody className='bg-white'>
        {resources.map(resource => (
          <tr key={resource} className='border-b border-gray-200'>
            <td className='whitespace-nowrap'>
              <a
                href={`/destinations/${destinationId}/${resource}`}
                className='block hover:bg-gray-100'
              >
                <div className='py-4'>
                  <div className='flex items-center justify-between'>
                    <p className='truncate text-sm font-medium text-gray-900'>
                      {resource}
                    </p>
                  </div>
                </div>
              </a>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

export default function DestinationDetail() {
  const router = useRouter()
  const destinationId = router.query.id

  const { admin, loading: adminLoading } = useAdmin()

  const { data: destination } = useSWR(`/api/destinations/${destinationId}`)
  const { data: auth } = useSWR('/api/users/self')
  const { data: { items: users } = {} } = useSWR('/api/users?limit=1000')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=1000')
  const { data: { items: grants } = {}, mutate } = useSWR(
    `/api/grants?resource=${destination?.name}&limit=1000`
  )
  const { data: { items: inherited } = {} } = useSWR(() =>
    parent(destination?.name)
      ? `/api/grants?resource=${parent(destination?.name)}&limit=1000`
      : null
  )
  const { data: { items: currentUserGrants } = {} } = useSWR(
    `/api/grants?user=${auth?.id}&resource=${destination?.name}&showInherited=1&limit=1000`
  )

  const { mutate: mutateCurrentUserGrants } = useSWRConfig()

  const [currentUserRoles, setCurrentUserRoles] = useState([])

  useEffect(() => {
    mutateCurrentUserGrants(
      `/api/grants?user=${auth?.id}&resource=${destination?.name}&showInherited=1&limit=1000`
    )

    const roles = currentUserGrants
      ?.filter(g => g.resource !== 'infra')
      ?.map(ug => ug.privilege)
      .sort(sortByPrivilege)

    setCurrentUserRoles(roles)
  }, [grants, auth, destination, currentUserGrants, mutateCurrentUserGrants])

  const loading = [
    !adminLoading,
    auth,
    destination,
    users,
    groups,
    grants,
  ].some(x => !x)

  return (
    <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
      <Head>
        <title>{destination?.name} - Infra</title>
      </Head>
      <Breadcrumbs>
        <Link href='/destinations'>
          <a>Clusters</a>
        </Link>
        {destination?.name}
      </Breadcrumbs>
      {!loading && (
        <div className='px-4 sm:px-6 md:px-0'>
          <div className='flex min-h-0 flex-1 flex-col px-0 md:px-6 xl:px-0'>
            <div className='py-6 xl:flex xl:items-center xl:justify-between'>
              <div className='min-w-0 flex-1'>
                <div className='flex items-center'>
                  <div>
                    <div className='flex items-center'>
                      <h1 className='text-2xl font-bold leading-7 text-gray-900 sm:truncate sm:leading-9'>
                        {destination?.name}
                      </h1>
                    </div>
                    <dl className='mt-6 flex flex-col sm:flex-row sm:flex-wrap'>
                      <dt className='sr-only'>Number of namespace</dt>
                      <dd className='mt-3 flex items-center text-sm font-medium text-gray-500 sm:mr-6 sm:mt-0'>
                        {destination.resources ? (
                          <>
                            {destination.resources.length}{' '}
                            {destination.resources.length === 1
                              ? 'namespace'
                              : 'namespaces'}
                          </>
                        ) : (
                          '0 namespaces'
                        )}
                      </dd>
                      <dt className='sr-only'>Version</dt>
                      <dd className='flex items-center text-sm font-medium text-gray-500 sm:mr-6'>
                        {destination.version ? <>{destination.version}</> : '-'}
                      </dd>
                      <dt className='sr-only'>Version</dt>
                      <dd className='flex items-center text-sm font-medium text-gray-500 sm:mr-6'>
                        <div
                          className={`h-2 w-2 flex-none rounded-full ${
                            destination.connected
                              ? 'bg-green-500'
                              : 'bg-gray-600'
                          }`}
                        />
                        <span className='flex-none px-2 text-gray-500'>
                          {destination.connected ? 'Connected' : 'Disconnected'}
                        </span>
                      </dd>
                    </dl>
                  </div>
                </div>
              </div>
              <div className='mt-6 flex space-x-3 xl:mt-0 xl:ml-4'>
                {admin && destination?.id && (
                  <RemoveButton
                    onRemove={async () => {
                      await fetch(`/api/destinations/${destination?.id}`, {
                        method: 'DELETE',
                      })

                      router.replace('/destinations')
                    }}
                    modalTitle='Remove Cluster'
                    modalMessage={
                      <>
                        Are you sure you want to disconnect{' '}
                        <span className='font-bold'>{destination?.name}?</span>
                        <br />
                        Note: you must also uninstall the Infra Connector from
                        this cluster.
                      </>
                    }
                  >
                    Remove cluster
                  </RemoveButton>
                )}
              </div>
            </div>
          </div>
          <div className='mt-6 space-y-10'>
            {admin && (
              <div>
                <h2 className='text-md border-b border-gray-200 py-2 font-medium text-gray-500'>
                  Namespaces
                </h2>
                <NamespacesTable
                  resources={destination?.resources}
                  destinationId={destinationId}
                />
              </div>
            )}
            {admin && (
              <div>
                <h2 className='text-md border-b border-gray-200 py-2 font-medium text-gray-500'>
                  Access
                </h2>
                <div className='space-y-6'>
                  <div>
                    <AccessTable
                      grants={grants}
                      users={users}
                      groups={groups}
                      destination={destination}
                      onRemove={async groupId => {
                        await fetch(`/api/grants/${groupId}`, {
                          method: 'DELETE',
                        })
                        mutate({
                          items: grants.filter(x => x.id !== groupId),
                        })
                      }}
                      onChange={async (privilege, group) => {
                        if (privilege === group.privilege) {
                          return
                        }

                        const res = await fetch('/api/grants', {
                          method: 'POST',
                          body: JSON.stringify({
                            ...group,
                            privilege,
                          }),
                        })

                        // delete old grant
                        await fetch(`/api/grants/${group.id}`, {
                          method: 'DELETE',
                        })

                        mutate({
                          items: [
                            ...grants.filter(f => f.id !== group.id),
                            await res.json(),
                          ],
                        })
                      }}
                      inherited={inherited}
                    />
                  </div>
                  <GrantForm
                    roles={destination?.roles}
                    onSubmit={async ({ user, group, privilege }) => {
                      // don't add grants that already exist
                      if (
                        grants?.find(
                          g =>
                            g.user === user &&
                            g.group === group &&
                            g.privilege === privilege
                        )
                      ) {
                        return false
                      }

                      const res = await fetch('/api/grants', {
                        method: 'POST',
                        body: JSON.stringify({
                          user,
                          group,
                          privilege,
                          resource: destination?.name,
                        }),
                      })

                      mutate({ items: [...grants, await res.json()] })
                    }}
                  />
                </div>
              </div>
            )}
            {currentUserRoles && currentUserRoles?.length > 0 && (
              <div>
                <h2 className='text-md border-b border-gray-200 py-2 font-medium text-gray-500'>
                  Connect
                </h2>
                <ConnectSection
                  userID={auth?.id}
                  roles={currentUserRoles}
                  kind={destination?.kind}
                  resource={destination?.name}
                />
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

DestinationDetail.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
