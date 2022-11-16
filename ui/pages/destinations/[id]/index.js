import { useEffect, useState, useRef } from 'react'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'
import { useRouter } from 'next/router'
import Head from 'next/head'
import Link from 'next/link'

import useSWR from 'swr'
import dayjs from 'dayjs'
import {
  CheckIcon,
  XMarkIcon,
  ChevronDownIcon,
  ChevronRightIcon,
} from '@heroicons/react/24/outline'
import { Transition, Listbox, Disclosure } from '@headlessui/react'

import { useUser } from '../../../lib/hooks'
import {
  sortByRole,
  sortBySubject,
  descriptions,
  sortByName,
} from '../../../lib/grants'

import GrantForm from '../../../components/grant-form'
import RemoveButton from '../../../components/remove-button'
import Dashboard from '../../../components/layouts/dashboard'
import Loader from '../../../components/loader'

const OPTION_SELECT_ALL = 'select all'
const METADATA_STATUS_LABEL = 'Status'
const OPTION_REMOVE = 'remove'

function EditRoleMenu({
  roles,
  selectedRoles,
  onChange,
  onRemove,
  privileges,
}) {
  roles = roles || []
  roles = sortByRole(roles)

  const [referenceElement, setReferenceElement] = useState(null)
  const [popperElement, setPopperElement] = useState(null)
  let { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: 'bottom-end',
    modifiers: [
      {
        name: 'flip',
        enabled: false,
      },
      {
        name: 'offset',
        options: {
          offset: [0, 5],
        },
      },
    ],
  })

  return (
    <Listbox
      value={selectedRoles}
      onChange={v => {
        if (v.includes(OPTION_REMOVE)) {
          onRemove()
          return
        }

        if (selectedRoles.length === 1 && v.length === 0) {
          return
        }

        onChange(v)
      }}
      multiple
    >
      <div className='relative'>
        <Listbox.Button
          ref={setReferenceElement}
          className='relative w-48 cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-8 text-left text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'
        >
          <div className='flex space-x-1 truncate'>
            <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
              <ChevronDownIcon
                className='h-4 w-4 stroke-1 text-gray-700'
                aria-hidden='true'
              />
            </span>
            <span className='text-gray-700'>{privileges?.[0]}</span>
            {privileges.length > 1 && (
              <span className='font-medium'> + {privileges.length - 1}</span>
            )}
          </div>
        </Listbox.Button>
        {ReactDOM.createPortal(
          <Listbox.Options
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className='absolute z-10 w-48 overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'
          >
            <div className='max-h-64 overflow-auto'>
              {roles?.map(r => (
                <Listbox.Option
                  key={r}
                  className={({ active }) =>
                    `${
                      active ? 'bg-gray-100' : ''
                    } select-none py-2 px-3 hover:cursor-pointer`
                  }
                  value={r}
                >
                  {({ selected }) => (
                    <div className='flex flex-row'>
                      <div className='flex flex-1 flex-col'>
                        <div className='flex justify-between py-0.5 font-medium'>
                          {r}
                          {selected && (
                            <CheckIcon
                              className='h-3 w-3 text-gray-900'
                              aria-hidden='true'
                            />
                          )}
                        </div>
                        <div className='text-3xs text-gray-600'>
                          {descriptions[r]}
                        </div>
                      </div>
                    </div>
                  )}
                </Listbox.Option>
              ))}
            </div>
            <Listbox.Option
              className={({ active }) =>
                `${
                  active ? 'bg-gray-50' : 'bg-white'
                } group flex w-full items-center border-t border-gray-100 px-2 py-1.5 text-xs font-medium text-red-500 hover:cursor-pointer`
              }
              value={OPTION_REMOVE}
            >
              <div className='flex flex-row items-center py-0.5'>
                <XMarkIcon className='mr-1 mt-px h-3.5 w-3.5' /> Remove access
              </div>
            </Listbox.Option>
          </Listbox.Options>,
          document.querySelector('body')
        )}
      </div>
    </Listbox>
  )
}

function GrantCell({ grantsList, grant, destination, onRemove, onUpdate }) {
  const checkbox = useRef()
  const [checked, setChecked] = useState(false)
  const [selectedNamespaces, setSelectedNamespaces] = useState([])

  const destinationPrivileges = grant.resourcePrivilegeMap.get(destination.name)

  const namespacesPrivilegeMap = new Map(
    Array.from(grant.resourcePrivilegeMap).filter(([key]) => {
      if (key.includes('.')) {
        return true
      }

      return false
    })
  )

  useEffect(() => {
    setChecked(
      selectedNamespaces.length === namespacesPrivilegeMap.size &&
        selectedNamespaces.length !== 0
    )
  }, [selectedNamespaces])

  function handleRemove(resource) {
    const deleteGrantIdList = grantsList
      .filter(g => g.resource === resource)
      .filter(g => g.user === grant.user)
      .filter(g => g.group === grant.group)
      .map(g => g.id)

    onRemove(deleteGrantIdList)
  }

  function handleUpdate(newPrivilege, selectedPrivilege, resource) {
    // update to add roles
    if (newPrivilege.length > selectedPrivilege.length) {
      const newRole = newPrivilege.filter(x => !selectedPrivilege.includes(x))
      onUpdate(newRole, resource)
    } else {
      // update to delete roles
      const removeRoles = selectedPrivilege.filter(
        x => !newPrivilege.includes(x)
      )

      const deleteGrantIdList = grantsList
        .filter(g => g.resource === resource)
        .filter(g => g.user === grant.user)
        .filter(g => g.group === grant.group)
        .filter(g => removeRoles.includes(g.privilege))
        .map(g => g.id)
      onRemove(deleteGrantIdList)
    }
  }

  return (
    <div className='py-1'>
      {/* Destination Resource */}
      {destinationPrivileges?.length > 0 && (
        <div className='flex items-center justify-between space-x-2 py-2'>
          <div className='text-xs font-medium text-black'>
            Cluster-wide access
          </div>
          <div className='item-center flex justify-between'>
            <EditRoleMenu
              roles={destination?.roles}
              selectedRoles={sortByRole(destinationPrivileges)}
              onChange={v => {
                handleUpdate(v, destinationPrivileges, destination.name)
              }}
              onRemove={() => {
                handleRemove(destination.name)
              }}
              privileges={sortByRole(destinationPrivileges)}
            />
          </div>
        </div>
      )}
      {/* Namespaces List */}
      {namespacesPrivilegeMap.size > 0 && (
        <div className='py-2'>
          <Disclosure defaultOpen={destinationPrivileges === undefined}>
            {({ open }) => (
              <>
                <div className='mb-2 flex items-center justify-between'>
                  <div className='flex items-center space-x-2'>
                    <input
                      type='checkbox'
                      className='h-4 w-4 rounded border-gray-300 text-blue-600 hover:cursor-pointer focus:ring-blue-500'
                      ref={checkbox}
                      checked={checked}
                      onChange={() => {
                        setSelectedNamespaces(
                          checked ? [] : [...namespacesPrivilegeMap.keys()]
                        )
                        setChecked(!checked)
                      }}
                    />
                    <Disclosure.Button className='w-full'>
                      <span className='flex items-center text-xs font-medium text-gray-500'>
                        {`Namespaces (${namespacesPrivilegeMap.size})`}
                        <ChevronRightIcon
                          className={`${
                            open ? 'rotate-90 transform' : ''
                          } ml-1 h-3 w-3 text-gray-500`}
                        />
                      </span>
                    </Disclosure.Button>
                  </div>
                  <button
                    className='rounded-md px-4 py-2 text-2xs font-medium text-red-500 hover:bg-red-100 disabled:cursor-not-allowed disabled:bg-white disabled:opacity-30'
                    type='button'
                    onClick={() => {
                      selectedNamespaces.map(namespace =>
                        handleRemove(namespace)
                      )
                      setSelectedNamespaces([])
                    }}
                    disabled={selectedNamespaces.length === 0}
                  >
                    <div className='flex flex-row items-center'>
                      <XMarkIcon className='mr-1 mt-px h-3.5 w-3.5' />
                      Remove selected
                    </div>
                  </button>
                </div>

                <Transition show={open}>
                  <Disclosure.Panel static>
                    <div className='space-y-2 pt-2'>
                      {[...namespacesPrivilegeMap.keys()]
                        .sort((a, b) => a.localeCompare(b))
                        .map(resource => {
                          const privileges =
                            namespacesPrivilegeMap.get(resource)

                          return (
                            <div
                              className='flex items-center justify-between'
                              key={resource}
                            >
                              <input
                                type='checkbox'
                                className='h-4 w-4 rounded border-gray-300 text-blue-600 hover:cursor-pointer focus:ring-blue-500 sm:left-6'
                                checked={selectedNamespaces.includes(resource)}
                                onChange={e => {
                                  setSelectedNamespaces(
                                    e.target.checked
                                      ? [...selectedNamespaces, resource]
                                      : selectedNamespaces.filter(
                                          r => r !== resource
                                        )
                                  )
                                }}
                              />
                              {resource && (
                                <div className='block w-1/2 truncate py-2 px-4 text-xs font-medium text-gray-900'>
                                  {resource.split('.').pop()}
                                </div>
                              )}
                              <EditRoleMenu
                                roles={destination?.roles}
                                selectedRoles={sortByRole(privileges)}
                                onChange={v => {
                                  handleUpdate(v, privileges, resource)
                                }}
                                onRemove={() => {
                                  handleRemove(resource)
                                }}
                                resource={resource}
                                privileges={sortByRole(privileges)}
                              />
                            </div>
                          )
                        })}
                    </div>
                  </Disclosure.Panel>
                </Transition>
              </>
            )}
          </Disclosure>
        </div>
      )}
    </div>
  )
}

function AccessTable({
  grants,
  users,
  groups,
  destination,
  onUpdate,
  onRemove,
}) {
  const grantsSubject = [...new Set(grants?.map(g => g.user || g.group))]
  let grantsList = []
  grantsSubject.forEach(subject => {
    let type = 'user'
    const grantArray = grants.filter(g => {
      if (g.group === subject) {
        type = 'group'
      }
      return g.user === subject || g.group === subject
    })

    const resourcePrivilegeMap = new Map()
    grantArray.forEach(g => {
      if (resourcePrivilegeMap.has(g.resource)) {
        resourcePrivilegeMap.set(g.resource, [
          ...resourcePrivilegeMap.get(g.resource),
          g.privilege,
        ])
      } else {
        resourcePrivilegeMap.set(g.resource, [g.privilege])
      }
    })

    const name =
      users?.find(u => u.id === subject)?.name ||
      groups?.find(g => g.id === subject)?.name

    if (grantArray.length === 1) {
      grantArray[0].resourcePrivilegeMap = resourcePrivilegeMap
      grantArray[0].name = name
      grantsList = [...grantsList, ...grantArray]
    } else {
      grantsList.push({
        name,
        [type]: subject,
        id: grantArray.map(g => g.id),
        resourcePrivilegeMap,
      })
    }
  })

  return (
    <div className='overflow-x-auto rounded-lg border border-gray-200/75'>
      <table className='w-full text-sm text-gray-600'>
        <thead className='border-b border-gray-200/75 bg-zinc-50/50 text-xs text-gray-500'>
          <tr>
            <th
              scope='col'
              className='py-2 px-5 text-left font-medium  sm:pl-6'
            >
              Group or user
            </th>
          </tr>
        </thead>
        <tbody className='divide-y divide-gray-200 bg-white'>
          {grantsList
            ?.sort(sortByName)
            ?.sort(sortBySubject)
            .map(grant => (
              <tr key={grant.user || grant.group}>
                <td className='w-[60%] whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6'>
                  <div className='flex w-[60%] flex-col truncate'>
                    <div className='text-sm font-medium text-gray-700'>
                      {grant.name}
                    </div>
                    <div className='text-2xs text-gray-500'>
                      {users?.find(u => u.id === grant.user) && 'User'}
                      {groups?.find(g => g.id === grant.group)?.name && 'Group'}
                    </div>
                  </div>
                </td>
                <td className='w-[35%] whitespace-nowrap px-3 py-4 text-sm text-gray-500'>
                  <GrantCell
                    grantsList={grants}
                    grant={grant}
                    destination={destination}
                    onRemove={grantsIdList => onRemove(grantsIdList)}
                    onUpdate={(newPrivilege, resource) =>
                      onUpdate(newPrivilege, grant.user, grant.group, resource)
                    }
                  />
                </td>
              </tr>
            ))}
        </tbody>
      </table>
      {grantsList && grantsList.length === 0 && (
        <div className='flex justify-center py-5 text-sm text-gray-500'>
          No data
        </div>
      )}
      {!grantsList && (
        <div className='flex w-full justify-center'>
          <Loader className='h-12 w-12' />
        </div>
      )}
    </div>
  )
}

function SelfAccessTable({ destination, grants }) {
  const resourcesList = [...new Set(grants?.map(g => g.resource))]
  const resourcesRoleList = []
  resourcesList.forEach(resource => {
    const roles = grants
      .filter(g => g.resource === resource)
      .map(g => g.privilege)

    resourcesRoleList.push({
      resource,
      roles,
    })
  })

  return (
    <div className='overflow-x-auto rounded-lg border border-gray-200/75'>
      <table className='w-full text-sm text-gray-600'>
        <thead className='border-b border-gray-200/75 bg-zinc-50/50 text-xs text-gray-500'>
          <tr>
            <th scope='col' className='py-2 px-5 text-left font-medium sm:pl-6'>
              Resource
            </th>
            <th scope='col' className='py-2 px-5 text-left font-medium sm:pl-6'>
              Roles
            </th>
          </tr>
        </thead>
        <tbody className='divide-y divide-gray-200 bg-white'>
          {resourcesRoleList.map(item => (
            <tr key={item.resource}>
              <td className='w-[30%] whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6'>
                {item.resource.split('.').pop() === destination
                  ? 'Cluster'
                  : item.resource.split('.').pop()}
              </td>
              <td className='w-[50%] whitespace-nowrap px-5 py-4 text-sm text-gray-500'>
                {item.roles.join(', ')}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {resourcesRoleList && resourcesRoleList.length === 0 && (
        <div className='flex justify-center py-5 text-sm text-gray-500'>
          You have no access to this cluster
        </div>
      )}
      {!resourcesRoleList && (
        <div className='flex w-full justify-center'>
          <Loader className='h-10 w-10' />
        </div>
      )}
    </div>
  )
}

function NamespacesDropdownMenu({
  selectedResources,
  setSelectedResources,
  resources,
}) {
  const [referenceElement, setReferenceElement] = useState(null)
  const [popperElement, setPopperElement] = useState(null)
  let { styles, attributes, update } = usePopper(
    referenceElement,
    popperElement,
    {
      placement: 'bottom-end',
      modifiers: [
        {
          name: 'flip',
          enabled: false,
        },
        {
          name: 'offset',
          options: {
            offset: [0, 5],
          },
        },
      ],
    }
  )

  return (
    <div className='relative'>
      <Listbox
        value={selectedResources}
        onChange={v => {
          if (v.includes(OPTION_SELECT_ALL)) {
            if (selectedResources.length !== resources.length) {
              setSelectedResources([...resources])
            } else {
              setSelectedResources([])
            }
            return
          }

          setSelectedResources(v)

          update()
        }}
        multiple
      >
        <div className='relative'>
          <Listbox.Button
            ref={setReferenceElement}
            className='relative w-48 cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-8 text-left text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'
          >
            <div className='flex space-x-1 truncate'>
              <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
                <ChevronDownIcon
                  className='h-4 w-4 stroke-1 text-gray-700'
                  aria-hidden='true'
                />
              </span>
              <span className='text-gray-700'>
                {selectedResources.length > 0
                  ? selectedResources[0]
                  : 'Select namespaces'}
              </span>
              {selectedResources.length - 1 > 0 && (
                <span> + {selectedResources.length - 1}</span>
              )}
            </div>
          </Listbox.Button>
          {ReactDOM.createPortal(
            <Listbox.Options
              ref={setPopperElement}
              style={styles.popper}
              {...attributes.popper}
              className='absolute z-10 w-48 overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'
            >
              <div className='max-h-64 overflow-auto'>
                {resources?.map(r => (
                  <Listbox.Option
                    key={r}
                    className={({ active }) =>
                      `${
                        active ? 'bg-gray-100' : ''
                      } select-none py-2 px-3 hover:cursor-pointer`
                    }
                    value={r}
                  >
                    {({ selected }) => (
                      <div className='flex flex-row'>
                        <div className='flex flex-1 flex-col'>
                          <div className='flex justify-between py-0.5 font-medium'>
                            {r}
                            {selected && (
                              <CheckIcon
                                className='h-3 w-3 text-gray-900'
                                aria-hidden='true'
                              />
                            )}
                          </div>
                        </div>
                      </div>
                    )}
                  </Listbox.Option>
                ))}
              </div>
              {resources.length > 1 && (
                <Listbox.Option
                  className={({ active }) =>
                    `${
                      active ? 'bg-gray-50' : 'bg-white'
                    } group flex w-full items-center border-t border-gray-100 px-2 py-1.5 text-xs font-medium text-blue-500 hover:cursor-pointer`
                  }
                  value={OPTION_SELECT_ALL}
                >
                  <div className='flex flex-row items-center py-0.5'>
                    {selectedResources.length !== resources.length
                      ? 'Select all'
                      : 'Reset'}
                  </div>
                </Listbox.Option>
              )}
            </Listbox.Options>,
            document.querySelector('body')
          )}
        </div>
      </Listbox>
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
  const { data: { items: currentUserAllGrants } = {} } = useSWR(
    `/api/grants?user=${user?.id}&destination=${destination?.name}`
  )

  const [selectedResources, setSelectedResources] = useState([])

  const metadata = [
    { label: 'ID', value: destination?.id, font: 'font-mono' },
    { label: '# of namespaces', value: destination?.resources.length },
    {
      label: METADATA_STATUS_LABEL,
      value: destination?.connected
        ? destination?.connection.url === ''
          ? 'Pending'
          : 'Connected'
        : 'Disconnected',
      style: destination?.connected
        ? destination?.connection.url === ''
          ? 'bg-yellow-100 text-yellow-800'
          : 'bg-green-100 text-green-800'
        : 'bg-gray-100 text-gray-800',
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
            <Link
              href='/destinations'
              className='text-gray-500/75 hover:text-gray-600'
            >
              Infrastructure
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
                {g.label !== METADATA_STATUS_LABEL && (
                  <span
                    className={`text-sm ${
                      g.font ? g.font : 'font-medium'
                    } text-gray-800`}
                  >
                    {g.value}
                  </span>
                )}
                {g.label === METADATA_STATUS_LABEL && (
                  <span
                    className={`${g.style} inline-flex items-center rounded-full px-2.5 py-px text-2xs font-medium`}
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
              <div className='flex flex-col space-y-2'>
                <div>
                  <h3 className='mb-3 text-sm font-medium'>
                    Grant access to{' '}
                    <span className='font-bold'>
                      {selectedResources.length > 0 ? (
                        selectedResources.length > 5 ? (
                          <span>
                            {selectedResources.slice(0, 5).join(', ')} ... +{' '}
                            {selectedResources.length - 5}
                          </span>
                        ) : (
                          selectedResources.join(', ')
                        )
                      ) : (
                        'cluster'
                      )}
                    </span>
                  </h3>{' '}
                  <GrantForm
                    roles={destination?.roles}
                    selectedResources={selectedResources}
                    grants={grants}
                    onSubmit={async ({
                      user,
                      group,
                      privilege,
                      selectedResources,
                    }) => {
                      // don't add grants that already exist
                      if (selectedResources.length === 0) {
                        if (
                          grants?.find(
                            g =>
                              g.user === user &&
                              g.group === group &&
                              g.privilege === privilege &&
                              g.resource === destination?.namn
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
                      } else {
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
                        setSelectedResources([])
                      }
                    }}
                  />
                </div>
                {destination?.resources.length > 0 && (
                  <div>
                    <Disclosure>
                      {({ open }) => (
                        <>
                          <Disclosure.Button className='w-full'>
                            <span className='flex items-center text-xs font-medium text-gray-500'>
                              <ChevronRightIcon
                                className={`${
                                  open ? 'rotate-90 transform' : ''
                                } mr-1 h-3 w-3 text-gray-500`}
                              />
                              Advanced
                            </span>
                          </Disclosure.Button>
                          <Transition show={open}>
                            <Disclosure.Panel static>
                              <div className='flex items-center space-x-4 px-4'>
                                <p className='text-xs text-gray-900'>
                                  Limit access to namespaces:
                                </p>
                                <NamespacesDropdownMenu
                                  selectedResources={selectedResources}
                                  setSelectedResources={setSelectedResources}
                                  resources={destination?.resources}
                                />
                              </div>
                            </Disclosure.Panel>
                          </Transition>
                        </>
                      )}
                    </Disclosure>
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
      {!isAdmin && (
        <SelfAccessTable
          destination={destination?.name}
          grants={currentUserAllGrants}
        />
      )}
    </div>
  )
}

DestinationDetail.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
