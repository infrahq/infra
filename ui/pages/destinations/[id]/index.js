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
  ChevronUpIcon,
  PlusIcon,
} from '@heroicons/react/outline'
import { Popover, Transition, Listbox, Disclosure } from '@headlessui/react'
import dayjs from 'dayjs'

import { useUser } from '../../../lib/hooks'
import { sortByPrivilege } from '../../../lib/grants'

import AccessTable from '../../../components/access-table'
import GrantForm from '../../../components/grant-form'
import RemoveButton from '../../../components/remove-button'
import Dashboard from '../../../components/layouts/dashboard'

function NamespacesGrantAccessForm({ children }) {
  return (
    <Disclosure>
      {({ open }) => (
        <>
          <Disclosure.Button className='w-full'>
            <span className='flex items-center text-xs font-medium text-gray-500 '>
              <ChevronUpIcon
                className={`${
                  open ? 'rotate-180 transform' : ''
                } h-4 w-4 text-gray-500 duration-300 ease-in`}
              />
              More Options
            </span>
          </Disclosure.Button>
          <Transition
            show={open}
            enter='ease-out duration-1000'
            enterFrom='opacity-0'
            enterTo='opacity-100'
            leave='ease-in duration-300'
            leaveFrom='opacity-100'
            leaveTo='opacity-0'
          >
            <Disclosure.Panel static>{children}</Disclosure.Panel>
          </Transition>
        </>
      )}
    </Disclosure>
  )
}

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
              <div className='flex flex-col space-y-4'>
                <div>
                  <h3 className='mb-3 text-sm font-medium'>
                    Grant access to <span className='font-bold'>cluster</span>
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
                            g.privilege === privilege &&
                            g.resource === `${destination?.name}`
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
                </div>
                {destination?.resources.length > 0 && (
                  <div>
                    <NamespacesGrantAccessForm>
                      <div className='pt-2'>
                        <h3 className='mb-3 text-sm font-medium'>
                          Grant access to{' '}
                          <span className='font-bold'>namespaces</span>
                        </h3>
                        <GrantForm
                          roles={destination?.roles.filter(
                            r => r != 'cluster-admin'
                          )}
                          grants={grants}
                          resources={destination?.resources}
                          onSubmit={async ({
                            user,
                            group,
                            privilege,
                            selectedResources,
                          }) => {
                            const promises = selectedResources.map(
                              async resource => {
                                // // don't add grants that already exist
                                if (
                                  grants?.find(
                                    g =>
                                      g.user === user &&
                                      g.group === group &&
                                      g.privilege === privilege &&
                                      g.resource ===
                                        `${destination?.name}.${resource}`
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
                                    resource: `${destination?.name}.${resource}`,
                                  }),
                                })
                              }
                            )

                            await Promise.all(promises)
                            mutate()
                          }}
                        />
                      </div>
                    </NamespacesGrantAccessForm>
                  </div>
                )}
              </div>
            </div>
            <AccessTable
              grants={grants}
              users={users}
              groups={groups}
              destination={destination}
              onUpdate={async (privileges, user, group, resource) => {
                const promises = privileges.map(
                  async privilege =>
                    await fetch('/api/grants', {
                      method: 'POST',
                      body: JSON.stringify({
                        user,
                        group,
                        privilege,
                        resource,
                      }),
                    })
                )

                await Promise.all(promises)
                mutate()
              }}
              onRemove={async grantsIdList => {
                const promises = grantsIdList.map(
                  async id =>
                    await fetch(`/api/grants/${id}`, {
                      method: 'DELETE',
                    })
                )

                await Promise.all(promises)
                mutate()
              }}
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
