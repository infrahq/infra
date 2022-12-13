import Head from 'next/head'
import { useEffect, useState, Fragment, useRef } from 'react'

import useSWR from 'swr'
import dayjs from 'dayjs'
import {
  TrashIcon,
  CheckIcon,
  ChevronDownIcon,
} from '@heroicons/react/24/outline'
import { Dialog, Transition, Combobox, Listbox } from '@headlessui/react'

import { useUser } from '../../lib/hooks'
import { descriptions, sortByRole } from '../../lib/grants'

import Dashboard from '../../components/layouts/dashboard'
import Table from '../../components/table'

const OPTION_SELECT_ALL = 'select all'

function CreateAccessDialog({ setOpen, grants, onCreated = () => {} }) {
  const { data: { items: users } = { items: [] } } = useSWR(
    '/api/users?limit=1000'
  )
  const { data: { items: groups } = { items: [] } } = useSWR(
    '/api/groups?limit=1000'
  )
  const { data: { items: resources } = { items: [] } } = useSWR(
    '/api/destinations?limit=1000'
  )

  const identityButton = useRef()
  const resourceButton = useRef()

  const [query, setQuery] = useState('')
  const [resourceQuery, setResourceQuery] = useState('')
  const [selected, setSelected] = useState(null)
  const [options, setOptions] = useState([])
  const [selectedResource, setSelectedResource] = useState(null)
  const [resourcesOptions, setResourcesOptions] = useState([])
  const [selectedNamespaces, setSelectedNamespaces] = useState([])
  const [selectedRoles, setSelectedRoles] = useState([])
  const [roles, setRoles] = useState([])

  useEffect(() => {
    if (users && groups) {
      const optionsList = [
        ...(groups?.map(g => ({ ...g, group: true })) || []),
        ...(users?.map(u => ({ ...u, user: true })) || []),
      ]

      setOptions(
        optionsList.filter(s =>
          s?.name?.toLowerCase()?.includes(query.toLowerCase())
        )
      )
    }
  }, [users, groups, grants, query])

  useEffect(() => {
    setResourcesOptions(
      resources.filter(r =>
        r?.name?.toLowerCase()?.includes(resourceQuery.toLowerCase())
      )
    )
  }, [resourceQuery])

  useEffect(() => setSelectedResource(resources[0]), [resources])

  useEffect(() => {
    setRoles(sortByRole(selectedResource?.roles))
  }, [selectedResource])

  useEffect(() => {
    if (roles.length > 0) {
      setSelectedRoles([roles[0]])
    }
  }, [roles])

  async function onSubmit(e) {
    e.preventDefault()

    setErrors({})
    setError('')

    try {
      const GrantsToAdd = []
      if (selectedNamespaces.length === 0) {
        selectedRoles.forEach(role => {
          GrantsToAdd.push({
            user: selected.user && selected.id,
            group: selected.group && selected.id,
            privilege: role,
            resource: selectedResource.name,
          })
        })
      } else {
        selectedNamespaces.forEach(resource => {
          selectedRoles.forEach(role => {
            GrantsToAdd.push({
              user: selected.user && selected.id,
              group: selected.group && selected.id,
              privilege: role,
              resource: `${selectedResource.name}.${resource}`,
            })
          })
        })
      }

      const newGrants = await fetch('/api/grants', {
        method: 'PATCH',
        body: JSON.stringify({ GrantsToAdd }),
      })

      onCreated(newGrants)
      setOpen(false)
    } catch (e) {
      console.error(e)

      return false
    }
  }

  return (
    <div className='w-full 2xl:m-auto'>
      <h1 className='py-1 font-display text-lg font-medium'>Create access</h1>
      <h3 className='mt-3 text-sm font-medium'>
        Grant access to{' '}
        <span className='font-bold'>
          {selectedNamespaces.length > 0 ? (
            selectedNamespaces.length > 5 ? (
              <span>
                {selectedNamespaces.slice(0, 5).join(', ')} ... +{' '}
                {selectedNamespaces.length - 5}
              </span>
            ) : (
              selectedNamespaces.join(', ')
            )
          ) : selectedResource?.kind === 'kubernetes' ? (
            'cluster'
          ) : (
            'ssh'
          )}
        </span>
      </h3>
      <form className='flex flex-col space-y-4' onSubmit={onSubmit}>
        <div className='mb-4 flex flex-col space-y-4'>
          <div className='mt-4 space-y-1'>
            <label className='text-2xs font-medium text-gray-700'>
              User or group
            </label>
            <Combobox
              as='div'
              value={selected?.name || ''}
              onChange={setSelected}
            >
              <Combobox.Input
                className={`block w-full rounded-md border-gray-300 text-xs shadow-sm focus:border-blue-500 focus:ring-blue-500`}
                placeholder='Enter group or user'
                onChange={e => {
                  setQuery(e.target.value)
                  if (e.target.value.length === 0) {
                    setSelected(null)
                  }
                }}
              />
              {options?.length > 0 && (
                <div className='relative'>
                  <Combobox.Options className='absolute z-50 mt-2 max-h-60 w-full origin-top-right divide-y divide-gray-100 overflow-auto rounded-md bg-white text-xs shadow-lg shadow-gray-300/20 ring-1 ring-black ring-opacity-5 focus:outline-none'>
                    {options?.map(f => (
                      <Combobox.Option
                        key={f.id}
                        value={f}
                        className={({ active }) =>
                          `relative cursor-default select-none py-[7px] px-3 ${
                            active ? 'bg-gray-50' : ''
                          }`
                        }
                      >
                        <div className='flex flex-row'>
                          <div className='flex min-w-0 flex-1 flex-col'>
                            <div className='flex justify-between py-0.5 font-medium'>
                              <span className='truncate' title={f.name}>
                                {f.name}
                              </span>
                              {selected && selected.id === f.id && (
                                <CheckIcon
                                  data-testid='selected-icon'
                                  className='h-3 w-3 stroke-1 text-gray-600'
                                  aria-hidden='true'
                                />
                              )}
                            </div>
                            <div className='text-3xs text-gray-500'>
                              {f.user ? 'User' : f.group ? 'Group' : ''}
                            </div>
                          </div>
                        </div>
                      </Combobox.Option>
                    ))}
                  </Combobox.Options>
                </div>
              )}
              <Combobox.Button className='hidden' ref={identityButton} />
            </Combobox>
          </div>
          <div className='relative mt-4 space-y-1'>
            <label className='text-2xs font-medium text-gray-700'>
              Resource
            </label>
            <Combobox
              as='div'
              value={selectedResource?.name || ''}
              onChange={setSelectedResource}
            >
              <Combobox.Input
                className={`block w-full rounded-md border-gray-300 text-xs shadow-sm focus:border-blue-500 focus:ring-blue-500`}
                placeholder='Enter resource'
                onChange={e => {
                  setResourceQuery(e.target.value)
                  if (e.target.value.length === 0) {
                    setSelectedResource(null)
                  }
                }}
              />
              {resourcesOptions?.length > 0 && (
                <div className='relative'>
                  <Combobox.Options className=' absolute z-50 mt-2 max-h-60 w-full origin-top-right divide-y divide-gray-100 overflow-auto rounded-md bg-white text-xs shadow-lg shadow-gray-300/20 ring-1 ring-black ring-opacity-5 focus:outline-none'>
                    {resourcesOptions?.map(f => (
                      <Combobox.Option
                        key={f.id}
                        value={f}
                        className={({ active }) =>
                          `relative cursor-default select-none py-[7px] px-3 ${
                            active ? 'bg-gray-50' : ''
                          }`
                        }
                      >
                        <div className='flex flex-row'>
                          <div className='flex min-w-0 flex-1 flex-col'>
                            <div className='flex justify-between py-0.5 font-medium'>
                              <span className='truncate' title={f.name}>
                                {f.name}
                              </span>
                              {selected && selected.id === f.id && (
                                <CheckIcon
                                  data-testid='selected-icon'
                                  className='h-3 w-3 stroke-1 text-gray-600'
                                  aria-hidden='true'
                                />
                              )}
                            </div>
                            <div className='text-3xs text-gray-500'>
                              {f.user ? 'User' : f.group ? 'Group' : ''}
                            </div>
                          </div>
                        </div>
                      </Combobox.Option>
                    ))}
                  </Combobox.Options>
                </div>
              )}
              <Combobox.Button className='hidden' ref={resourceButton} />
            </Combobox>
          </div>
          {selectedResource !== null && (
            <div className='relative mt-4 space-y-1'>
              <label className='text-2xs font-medium text-gray-700'>
                Namespaces
              </label>
              <Listbox
                value={selectedNamespaces}
                onChange={v => {
                  if (v.includes(OPTION_SELECT_ALL)) {
                    if (
                      selectedNamespaces.length !==
                      selectedResource?.resources?.length
                    ) {
                      setSelectedNamespaces([...selectedResource?.resources])
                    } else {
                      setSelectedNamespaces([])
                    }
                    return
                  }

                  setSelectedNamespaces(v)
                }}
                multiple
              >
                <div className='relative'>
                  <Listbox.Button className='relative w-full cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-8 text-left text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'>
                    <div className='flex space-x-1 truncate'>
                      <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
                        <ChevronDownIcon
                          className='h-4 w-4 stroke-1 text-gray-700'
                          aria-hidden='true'
                        />
                      </span>
                      <span className='text-gray-700'>
                        {selectedNamespaces.length > 0
                          ? selectedNamespaces.join(', ')
                          : 'Select namespaces'}
                      </span>
                    </div>
                  </Listbox.Button>
                  <Listbox.Options className='absolute z-10 w-full overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'>
                    <div className='max-h-64 overflow-auto'>
                      {selectedResource?.resources?.map(r => (
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
                    {selectedResource?.resources?.length > 1 && (
                      <Listbox.Option
                        className={({ active }) =>
                          `${
                            active ? 'bg-gray-50' : 'bg-white'
                          } group flex w-full items-center border-t border-gray-100 px-2 py-1.5 text-xs font-medium text-blue-500 hover:cursor-pointer`
                        }
                        value={OPTION_SELECT_ALL}
                      >
                        <div className='flex flex-row items-center py-0.5'>
                          {selectedNamespaces.length !==
                          selectedResource?.resources?.length
                            ? 'Select all'
                            : 'Reset'}
                        </div>
                      </Listbox.Option>
                    )}
                  </Listbox.Options>
                </div>
              </Listbox>
            </div>
          )}
          {roles?.length > 1 && (
            <div className='relative mt-4 space-y-1'>
              <label className='text-2xs font-medium text-gray-700'>
                Roles
              </label>
              <Listbox
                value={selectedRoles}
                onChange={v => {
                  if (selectedRoles.length === 1 && v.length === 0) {
                    return
                  }

                  const add = v.filter(x => !selectedRoles.includes(x))
                  const remove = selectedRoles.filter(x => !v.includes(x))
                  if (add.length) {
                    setSelectedRoles([...selectedRoles, ...add])
                  }
                  if (remove.length) {
                    setSelectedRoles(
                      selectedRoles.filter(x => !remove.includes(x))
                    )
                  }
                }}
                multiple
              >
                <div className='relative'>
                  <Listbox.Button className='relative w-full cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-8 text-left text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'>
                    <div className='flex space-x-1 truncate'>
                      <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
                        <ChevronDownIcon
                          className='h-4 w-4 stroke-1 text-gray-700'
                          aria-hidden='true'
                        />
                      </span>
                      <span className='text-gray-700'>
                        {selectedRoles.join(', ')}
                      </span>
                    </div>
                  </Listbox.Button>
                  <Listbox.Options className='absolute z-[100] w-full overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'>
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
                  </Listbox.Options>
                </div>
              </Listbox>
            </div>
          )}
        </div>
        <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
          <button
            type='button'
            onClick={() => setOpen(false)}
            className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-100'
          >
            Cancel
          </button>
          <button
            type='submit'
            disabled={!selected || !selectedResource}
            className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-30'
          >
            Add
          </button>
        </div>
      </form>
    </div>
  )
}

export default function AccessControl() {
  const { user, isAdmin } = useUser()

  const { data: { items: users } = {} } = useSWR('/api/users?limit=1000')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=1000')

  const { data: { items: allGrants } = {}, mutate } = useSWR(() =>
    isAdmin
      ? '/api/grants?limit=1000'
      : `/api/grants?user=${user.id}&limit=1000`
  )

  const [grants, setGrants] = useState({})
  const [openCreateAccess, setOpenCreateAccess] = useState(false)

  useEffect(() => {
    setGrants(
      allGrants
        ?.filter(g => g.resource !== 'infra')
        .map(g => {
          if (g.group) {
            return { ...g, type: 'group', identityId: g.group }
          }

          if (g.user) {
            return { ...g, type: 'user', identityId: g.user }
          }

          return g
        })
    )
  }, [allGrants])

  const columns = []
  if (isAdmin) {
    columns.push({
      header: <span>User / Group </span>,
      id: 'identity',
      accessorKey: 'identityId',
      cell: function Cell(info) {
        const name =
          users?.find(u => u.id === info.row.original.identityId)?.name ||
          groups?.find(g => g.id === info.row.original.identityId)?.name

        return (
          <div className='flex flex-col'>
            <div className='text-sm font-medium text-gray-700'>{name}</div>
            <div className='text-2xs text-gray-500'>
              {info.row.original.type}
            </div>
          </div>
        )
      },
    })
  }

  return (
    <div className='mb-10'>
      <Head>
        <title>Access Control - Infra</title>
      </Head>
      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 font-display text-xl font-medium'>
          Access Control
        </h1>
        {isAdmin && (
          <>
            <button
              onClick={() => setOpenCreateAccess(true)}
              className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
            >
              Create access
            </button>
            <Transition.Root show={openCreateAccess} as={Fragment}>
              <Dialog as='div' className='relative z-50' onClose={() => {}}>
                <Transition.Child
                  as={Fragment}
                  enter='ease-out duration-150'
                  enterFrom='opacity-0'
                  enterTo='opacity-100'
                  leave='ease-in duration-100'
                  leaveFrom='opacity-100'
                  leaveTo='opacity-0'
                >
                  <div className='fixed inset-0 bg-white bg-opacity-75 backdrop-blur-xl transition-opacity' />
                </Transition.Child>
                <div className='fixed inset-0 z-30 overflow-y-auto'>
                  <div className='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
                    <Transition.Child
                      as={Fragment}
                      enter='ease-out duration-150'
                      enterFrom='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                      enterTo='opacity-100 translate-y-0 sm:scale-100'
                      leave='ease-in duration-100'
                      leaveFrom='opacity-100 translate-y-0 sm:scale-100'
                      leaveTo='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                    >
                      <Dialog.Panel className='relative w-full transform rounded-xl border border-gray-100 bg-white p-8 text-left shadow-xl shadow-gray-300/10 transition-all sm:my-8 sm:max-w-md'>
                        <CreateAccessDialog
                          setOpen={setOpenCreateAccess}
                          grants={grants}
                          onCreated={() => {
                            mutate()
                          }}
                        />
                      </Dialog.Panel>
                    </Transition.Child>
                  </div>
                </div>
              </Dialog>
            </Transition.Root>
          </>
        )}
      </header>
      <Table
        data={grants}
        allowDelete={isAdmin}
        onDelete={async selectedIds => {
          const promises = selectedIds.map(
            async selectedId =>
              await fetch(`/api/grants/${selectedId}`, { method: 'DELETE' })
          )

          await Promise.all(promises)
          mutate()
        }}
        columns={[
          ...columns,
          {
            cell: info => <span>{info.getValue()}</span>,
            header: () => <span>Resource</span>,
            accessorKey: 'resource',
          },
          {
            cell: info => <span>{info.getValue()}</span>,
            header: <span>Role</span>,
            accessorKey: 'privilege',
          },
          {
            cell: info => (
              <div className='hidden sm:table-cell'>
                {info.getValue() ? dayjs(info.getValue()).fromNow() : '-'}
              </div>
            ),
            header: () => <span>Created</span>,
            accessorKey: 'created',
          },
          {
            id: 'delete',
            cell: function Cell(info) {
              return (
                isAdmin && (
                  <button
                    type='button'
                    onClick={async () => {
                      await fetch(`/api/grants/${info.row.original.id}`, {
                        method: 'DELETE',
                      })
                      mutate()
                    }}
                    className='group flex w-full items-center rounded-md bg-white px-2 py-1.5 text-right text-xs font-medium text-red-500 hover:text-red-500/50'
                  >
                    <TrashIcon className='mr-2 h-3.5 w-3.5' />
                    <span className='hidden sm:block'>Remove</span>
                  </button>
                )
              )
            },
          },
        ]}
      />
    </div>
  )
}

AccessControl.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
