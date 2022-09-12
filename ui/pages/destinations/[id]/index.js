import { useRouter } from 'next/router'
import useSWR, { useSWRConfig } from 'swr'
import { useEffect, useState, Fragment } from 'react'
import Head from 'next/head'
import Link from 'next/link'
import copy from 'copy-to-clipboard'
import {
  CheckIcon,
  DuplicateIcon,
  DownloadIcon,
  ChevronDownIcon,
} from '@heroicons/react/outline'
import { Popover, Transition } from '@headlessui/react'

import { useAdmin } from '../../../lib/admin'
import { sortByPrivilege } from '../../../lib/grants'

import Table from '../../../components/table'
import AccessTable from '../../../components/access-table'
import GrantForm from '../../../components/grant-form'
import RemoveButton from '../../../components/remove-button'
import Dashboard from '../../../components/layouts/dashboard'

function parent(resource = '') {
  const parts = resource.split('.')
  return parts.length > 1 ? parts[0] : null
}

function AccessCluster({ roles, resource }) {
  const [commandCopied, setCommandCopied] = useState(false)

  const command = `infra login ${window.location.host} \ninfra use ${resource} \nkubectl get pods`

  return (
    <div className='w-full flex-1'>
      <div className='mx-6 mt-4 mb-1 flex items-center justify-between text-sm'>
        <h1 className='flex items-center font-semibold text-black'>
          Access cluster
        </h1>
        <a
          target='_blank'
          href='https://infrahq.com/docs/install/install-infra-cli'
          className='flex items-center text-xs font-medium text-gray-800 hover:text-gray-400'
          rel='noreferrer'
        >
          <DownloadIcon className='mr-1 h-3.5 w-3.5' />
          Infra CLI
        </a>
      </div>
      <p className='mx-6 my-4 text-xs text-gray-800'>
        You have{' '}
        <span className='font-semibold text-black'>{roles.join(', ')}</span>{' '}
        access.
      </p>
      <div className='group relative mt-4 flex flex-1 flex-col'>
        <pre className='w-full flex-1 overflow-auto break-all bg-gray-100 p-6 py-5 text-2xs leading-normal text-gray-900'>
          {command}
        </pre>
        <button
          className={`absolute right-2 top-2 rounded-md border border-white/10 bg-white/5 px-2 py-2 text-white opacity-0 backdrop-blur-xl ${
            commandCopied ? 'opacity-100' : 'group-hover:opacity-100'
          }`}
          disabled={commandCopied}
          onClick={() => {
            copy(command)
            setCommandCopied(true)
            setTimeout(() => setCommandCopied(false), 2000)
          }}
        >
          {commandCopied ? (
            <CheckIcon className='h-4 w-4 stroke-1 text-green-500' />
          ) : (
            <DuplicateIcon className='h-4 w-4 stroke-1' />
          )}
        </button>
      </div>
    </div>
  )
}

const TAB_ACCESS = 'access'
const TAB_NAMESPACES = 'namespaces'

export default function DestinationDetail() {
  const router = useRouter()
  const destinationId = router.query.id

  const { admin } = useAdmin()

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

  const tab = router.query.tab || TAB_ACCESS

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

  return (
    <div className='mb-10'>
      <Head>
        <title>{destination?.name} - Infra</title>
      </Head>
      <header className='mt-6 mb-12 flex items-center justify-between'>
        <h1 className='flex py-1 text-xl font-medium'>
          <Link href='/destinations'>
            <a className='text-gray-500/75 hover:text-gray-600'>Clusters</a>
          </Link>{' '}
          <span className='mx-3 font-light text-gray-400'> / </span>{' '}
          <div className='flex'>
            <div className='mr-2 flex h-8 w-8 flex-none items-center justify-center rounded-md border border-gray-200'>
              <img
                alt='kubernetes icon'
                className='h-[18px]'
                src={`/kubernetes.svg`}
              />
            </div>
            {destination?.name}
          </div>
        </h1>
        <div className='flex space-x-2'>
          {currentUserRoles && currentUserRoles?.length > 0 && (
            <Popover className='relative'>
              <Popover.Button className='inline-flex items-center rounded-md border border-transparent bg-blue-500 px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-blue-600'>
                Access cluster
                <ChevronDownIcon className='ml-1 h-4 w-4' />
              </Popover.Button>
              <Transition
                as={Fragment}
                enter='transition ease-out duration-100 origin-top-right'
                enterFrom='transform opacity-0 scale-90 translate-y-0'
                enterTo='transform opacity-100 scale-100 translate-y-1'
                leave='transition ease-in duration-75 origin-top-right'
                leaveFrom='transform opacity-100 scale-100 translate-y-1'
                leaveTo='transform opacity-0 scale-90 translate-y-0'
              >
                <Popover.Panel className='absolute right-0 z-10 my-2 flex w-80 overflow-hidden rounded-xl border border-gray-300 bg-white text-white shadow-2xl shadow-black/40'>
                  <AccessCluster
                    userID={auth?.id}
                    roles={currentUserRoles}
                    kind={destination?.kind}
                    resource={destination?.name}
                  />
                </Popover.Panel>
              </Transition>
            </Popover>
          )}
          {admin && (
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
                  Are you sure you want to remove{' '}
                  <span className='font-bold'>{destination?.name}?</span>
                  <br />
                  Note: you must also uninstall the Infra Connector from this
                  cluster.
                </>
              }
            >
              Remove cluster
            </RemoveButton>
          )}
        </div>
      </header>

      {/* Tabs */}
      <div className='mb-6 border-b border-gray-200'>
        <nav className='-mb-px flex' aria-label='Tabs'>
          {[TAB_ACCESS, TAB_NAMESPACES].map(t => (
            <Link
              key={t}
              href={{
                pathname: `/destinations/${destination?.id}`,
                query: { ...router.query, tab: t },
              }}
            >
              <a
                className={`
                ${
                  tab === t
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-600'
                }
                 whitespace-nowrap border-b-2 py-2 px-5 text-sm font-medium capitalize transition-colors`}
                aria-current={tab.current ? 'page' : undefined}
              >
                {t}
              </a>
            </Link>
          ))}
        </nav>
      </div>

      {tab === TAB_ACCESS && (
        <div className='flex flex-col space-y-1'>
          <div className='max-w-md self-end'>
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
      )}

      {tab === TAB_NAMESPACES && (
        <Table
          data={destination?.resources}
          empty='No namespaces'
          href={row => `/destinations/${destination?.id}/${row.original}`}
          columns={[
            {
              id: 'name',
              cell: info => {
                return (
                  <span className='font-medium text-gray-700'>
                    {info.row.original}
                  </span>
                )
              },
              header: () => <span>Namespace</span>,
            },
          ]}
        />
      )}
    </div>
  )
}

DestinationDetail.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
