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
  PlusIcon,
} from '@heroicons/react/outline'
import { Popover, Transition, Listbox } from '@headlessui/react'
import dayjs from 'dayjs'

import { useUser } from '../../../lib/hooks'
import { sortByPrivilege } from '../../../lib/grants'

import Table from '../../../components/table'
import AccessTable from '../../../components/access-table'
import GrantForm from '../../../components/grant-form'
import RemoveButton from '../../../components/remove-button'
import Dashboard from '../../../components/layouts/dashboard'

function AccessCluster({ roles, resource }) {
  const [commandCopied, setCommandCopied] = useState(false)

  const command = `infra login ${window.location.host} \ninfra use ${resource} \nkubectl get pods`

  return (
    <div className='w-full flex-1'>
      <div className='mx-6 mt-4 mb-1 flex items-center justify-between text-sm'>
        <h1 className='flex items-center font-semibold'>Access cluster</h1>
        <a
          target='_blank'
          href='https://infrahq.com/docs/install/install-infra-cli'
          className='flex items-center text-xs font-medium text-gray-300 hover:text-gray-400'
          rel='noreferrer'
        >
          <DownloadIcon className='mr-1 h-3.5 w-3.5' />
          Infra CLI
        </a>
      </div>
      <p className='mx-6 my-4 text-xs text-gray-300'>
        You have{' '}
        <span className='font-semibold text-white'>{roles.join(', ')}</span>{' '}
        access.
      </p>
      <div className='group relative mt-4 flex flex-1 flex-col'>
        <pre className='w-full flex-1 overflow-auto break-all bg-zinc-900 p-6 pt-4 text-2xs leading-normal text-gray-300'>
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

function GrantAccessTypesMenu({
  destination,
  typeList,
  selectedList,
  onChange,
}) {
  return (
    <Listbox value={selectedList} onChange={v => onChange(v)} multiple>
      <div className='relative mt-1'>
        <Listbox.Button className='relative w-full cursor-default py-2 text-left text-sm hover:cursor-pointer'>
          <div className='flex items-center truncate font-semibold text-gray-500'>
            <PlusIcon className='mr-1 h-3 w-3' />
            <span>select resources</span>
          </div>
        </Listbox.Button>

        <Listbox.Options className='absolute z-10 mt-1 w-56 overflow-auto rounded-md bg-white text-sm shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none'>
          <div className='bg-gray-100'>
            <div className='py-2 px-3 text-xs font-bold text-gray-400'>
              cluster
            </div>
          </div>
          <div className='max-h-64 overflow-auto'>
            <Listbox.Option
              key={destination}
              className={({ active }) =>
                `${active ? 'bg-blue-600 text-white' : 'text-gray-900'}
                          relative cursor-pointer select-none py-2 pl-3 pr-9`
              }
              value={destination}
            >
              {({ selected, active }) => (
                <>
                  <span
                    className={`${selected ? 'font-semibold' : 'font-normal'}
                              block truncate`}
                  >
                    {destination}
                  </span>

                  {selected ? (
                    <span
                      className={`
                                ${active ? 'text-white' : 'text-blue-600'}
                                absolute inset-y-0 right-0 flex items-center pr-4`}
                    >
                      <CheckIcon className='h-5 w-5' aria-hidden='true' />
                    </span>
                  ) : null}
                </>
              )}
            </Listbox.Option>
            <div className='bg-gray-100'>
              <div className='py-2 px-3 text-xs font-bold text-gray-400'>
                namespaces
              </div>
            </div>
            {typeList.map(type => (
              <Listbox.Option
                key={type}
                className={({ active }) =>
                  `${active ? 'bg-blue-600 text-white' : 'text-gray-900'}
                          relative cursor-pointer select-none py-2 pl-3 pr-9`
                }
                value={type}
              >
                {({ selected, active }) => (
                  <>
                    <span
                      className={`${selected ? 'font-semibold' : 'font-normal'}
                              block truncate`}
                    >
                      {type}
                    </span>

                    {selected ? (
                      <span
                        className={`
                                ${active ? 'text-white' : 'text-blue-600'}
                                absolute inset-y-0 right-0 flex items-center pr-4`}
                      >
                        <CheckIcon className='h-5 w-5' aria-hidden='true' />
                      </span>
                    ) : null}
                  </>
                )}
              </Listbox.Option>
            ))}
          </div>
          <div className='border-t'>
            <div className='flex items-center justify-between py-2 px-3 text-xs font-medium text-blue-500'>
              <button
                className='disabled:cursor-not-allowed disabled:opacity-30'
                disabled={
                  selectedList.length === 1 &&
                  selectedList.includes(destination)
                }
                onClick={() => {
                  onChange([destination])
                }}
              >
                clear
              </button>
              <button
                className='disabled:cursor-not-allowed disabled:opacity-30'
                disabled={
                  selectedList.length === typeList.length + 1 &&
                  selectedList.sort().join(',') ===
                    [destination, ...typeList].sort().join(',')
                }
                onClick={() => {
                  onChange([destination, ...typeList])
                }}
              >
                select all
              </button>
            </div>
          </div>
        </Listbox.Options>
      </div>
    </Listbox>
  )
}

export default function DestinationDetail() {
  const router = useRouter()
  const destinationId = router.query.id

  const { data: destination } = useSWR(`/api/destinations/${destinationId}`)
  const { user, isAdmin } = useUser()
  const { data: { items: users } = {} } = useSWR('/api/users?limit=1000')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=1000')
  const { data: { items: grants } = {}, mutate } = useSWR(
    `/api/grants?destination=${destination?.name}`
  )
  const { data: { items: currentUserGrants } = {} } = useSWR(
    `/api/grants?user=${user?.id}&resource=${destination?.name}&showInherited=1&limit=1000`
  )

  const { mutate: mutateCurrentUserGrants } = useSWRConfig()

  const [currentUserRoles, setCurrentUserRoles] = useState([])
  const [grantAccessTypeLists, setGrantAccessTypeLists] = useState([])

  useEffect(() => {
    mutateCurrentUserGrants(
      `/api/grants?user=${user?.id}&resource=${destination?.name}&showInherited=1&limit=1000`
    )

    const roles = currentUserGrants
      ?.filter(g => g.resource !== 'infra')
      ?.map(ug => ug.privilege)
      .sort(sortByPrivilege)

    setCurrentUserRoles(roles)
  }, [grants, user, destination, currentUserGrants, mutateCurrentUserGrants])

  useEffect(() => {
    setGrantAccessTypeLists([destination?.name])
  }, [destination])

  const metadata = [
    { label: 'ID', value: destination?.id, font: 'font-mono' },
    { label: '# of namespaces', value: destination?.resources.length },
    {
      label: 'Status',
      value: destination?.connected
        ? destination?.connection.url === ''
          ? 'Pending'
          : 'Connected'
        : 'Disconnected',
      color: destination?.connected
        ? destination?.connection.url === ''
          ? 'yellow'
          : 'green'
        : 'gray',
    },
    {
      label: 'Created',
      value: destination?.created ? dayjs(destination?.created).fromNow() : '-',
    },
  ]

  return (
    <div className='mb-10'>
      <Head>
        <title>{destination?.name} - Infra</title>
      </Head>
      <header className='mt-6 mb-12 space-y-4'>
        <div className=' flex flex-col justify-between md:flex-row md:items-center'>
          <h1 className='flex max-w-[75%] truncate py-1 font-display text-xl font-medium'>
            <Link href='/destinations'>
              <a className='text-gray-500/75 hover:text-gray-600'>
                Infrastructure
              </a>
            </Link>{' '}
            <span className='mx-3 font-light text-gray-400'> / </span>{' '}
            <div className='flex truncate'>
              <div className='mr-2 flex h-8 w-8 flex-none items-center justify-center rounded-md border border-gray-200'>
                <img
                  alt='kubernetes icon'
                  className='h-[18px]'
                  src={`/kubernetes.svg`}
                />
              </div>
              <div className='flex items-center space-x-2'>
                <span className='truncate'>{destination?.name}</span>
                <div
                  className={`h-2 w-2 flex-none rounded-full border ${
                    destination?.connected
                      ? destination?.connection.url === ''
                        ? 'animate-pulse border-yellow-500 bg-yellow-500'
                        : 'border-teal-400 bg-teal-400'
                      : 'border-gray-200 bg-gray-200'
                  }`}
                />
              </div>
            </div>
          </h1>
          <div className='my-3 flex space-x-2 md:my-0'>
            {currentUserRoles && currentUserRoles?.length > 0 && (
              <Popover className='relative'>
                <Popover.Button className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-xs font-semibold text-blue-500 shadow-sm hover:text-blue-600'>
                  Access cluster
                  <ChevronDownIcon className='ml-1 h-4 w-4' />
                </Popover.Button>
                <Transition
                  as={Fragment}
                  enter='transition ease-out duration-100 origin-top-left md:origin-top-right'
                  enterFrom='transform opacity-0 scale-90 translate-y-0'
                  enterTo='transform opacity-100 scale-100 translate-y-1'
                  leave='transition ease-in duration-75 origin-top-left md:origin-top-right'
                  leaveFrom='transform opacity-100 scale-100 translate-y-1'
                  leaveTo='transform opacity-0 scale-90 translate-y-0'
                >
                  <Popover.Panel className='absolute left-0 z-10 flex w-80 overflow-hidden rounded-xl bg-black text-white shadow-2xl shadow-black/40 md:left-auto md:right-0'>
                    <AccessCluster
                      userID={user?.id}
                      roles={currentUserRoles}
                      kind={destination?.kind}
                      resource={destination?.name}
                    />
                  </Popover.Panel>
                </Transition>
              </Popover>
            )}
            {isAdmin && (
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
        </div>
        {destination && (
          <div className='flex flex-col border-t border-gray-100 sm:flex-row'>
            {metadata.map(g => (
              <div
                key={g.label}
                className='py-5 text-left sm:px-6 sm:first:pr-6 sm:first:pl-0'
              >
                <div className='text-2xs text-gray-400'>{g.label}</div>
                {g.label !== 'Status' && (
                  <span
                    className={`text-sm ${
                      g.font ? g.font : 'font-medium'
                    } text-gray-800`}
                  >
                    {g.value}
                  </span>
                )}
                {g.label === 'Status' && (
                  <span
                    className={`inline-flex items-center rounded-full bg-${g.color}-100 px-2.5 py-px text-2xs font-medium text-${g.color}-800`}
                  >
                    {g.value}
                  </span>
                )}
              </div>
            ))}
          </div>
        )}
      </header>
      {isAdmin && (
        <>
          <div className='my-5 flex flex-col space-y-4'>
            <div className='w-full rounded-lg border border-gray-200/75 px-5 py-3'>
              <h3 className='mb-3 text-sm font-medium'>
                Grant access to{' '}
                <span className='font-bold'>
                  {grantAccessTypeLists.join(', ')}
                </span>
              </h3>{' '}
              <GrantForm
                roles={destination?.roles}
                grants={grants}
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

                  await fetch('/api/grants', {
                    method: 'POST',
                    body: JSON.stringify({
                      user,
                      group,
                      privilege,
                      resource: destination?.name,
                    }),
                  })

                  mutate()
                }}
              />
              {destination && (
                <GrantAccessTypesMenu
                  destination={destination?.name}
                  typeList={destination?.resources}
                  selectedList={grantAccessTypeLists}
                  onChange={setGrantAccessTypeLists}
                />
              )}
            </div>
            <AccessTable
              grants={grants}
              users={users}
              groups={groups}
              destination={destination}
              onUpdate={async (privilege, user, group, resource) => {
                await fetch('/api/grants', {
                  method: 'POST',
                  body: JSON.stringify({
                    user,
                    group,
                    privilege,
                    resource,
                  }),
                })

                mutate()
              }}
              onRemove={async grantId => {
                await fetch(`/api/grants/${grantId}`, {
                  method: 'DELETE',
                })
                mutate()
              }}
              onChange={async (privilege, group) => {
                if (privilege === group.privilege) {
                  return
                }

                await fetch('/api/grants', {
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

                mutate()
              }}
            />
          </div>
          <div>
            <header className='mt-6 mb-3 flex'>
              <h1 className='font-display text-base font-medium'>Namespaces</h1>
            </header>
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
                  header: () => <span>Name</span>,
                },
              ]}
            />
          </div>
        </>
      )}
    </div>
  )
}

DestinationDetail.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
