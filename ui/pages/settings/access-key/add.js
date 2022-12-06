import Link from 'next/link'
import Head from 'next/head'
import { Fragment, useEffect, useState } from 'react'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'
import { useRouter } from 'next/router'
import useSWR from 'swr'

import {
  XMarkIcon,
  ChevronDownIcon,
  CheckIcon,
  DocumentDuplicateIcon,
} from '@heroicons/react/24/outline'
import { Transition, Listbox, Dialog, Popover } from '@headlessui/react'
import copy from 'copy-to-clipboard'
import moment from 'moment'

import { useUser } from '../../../lib/hooks'

import Dashboard from '../../../components/layouts/dashboard'
import Calendar from '../../../components/calendar'

const CUSTOM_TITLE = 'custom...'

const EXPIRATION_RATE = [
  { name: '30 days', value: '720h' },
  { name: '60 days', value: '1440h' },
  { name: '90 days', value: '2160h' },
  { name: '1 year', value: '8766h' },
  { name: CUSTOM_TITLE, value: '720h', custom: true },
]

function AccessKeyDialogContent({ accessKey }) {
  const [keyCopied, setKeyCopied] = useState(false)

  return (
    <div className='w-full 2xl:m-auto'>
      <h1 className='py-1 font-display text-lg font-medium'>Access Key</h1>
      <div className='space-y-4'>
        <div className='mb-2'>
          <p className='mt-1 text-sm text-gray-500'>
            Make sure to copy the access key now as you will not be able to see
            this again.
          </p>
        </div>
        <div className='group relative my-4 flex'>
          <pre className='w-full overflow-auto rounded-lg bg-gray-50 px-5 py-4 text-xs leading-normal text-gray-800'>
            {accessKey}
          </pre>
          <button
            className='absolute right-2 top-2 rounded-md border border-black/10 bg-white px-2 py-2 text-black/40 backdrop-blur-xl hover:text-black/70'
            type='button'
            onClick={() => {
              copy(accessKey)
              setKeyCopied(true)
              setTimeout(() => setKeyCopied(false), 2000)
            }}
          >
            {keyCopied ? (
              <CheckIcon className='h-4 w-4 text-green-500' />
            ) : (
              <DocumentDuplicateIcon className='h-4 w-4' />
            )}
          </button>
        </div>

        {/* Finish */}
        <div className='my-10 flex justify-end'>
          <Link
            href='/settings'
            className='flex-none items-center self-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'
          >
            Finish
          </Link>
        </div>
      </div>
    </div>
  )
}

function AccessKeyDialog({ accessKey, accessKeyDialogOpen }) {
  return (
    <Transition.Root show={accessKeyDialogOpen} as={Fragment}>
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
        <div className='fixed inset-0 z-10 overflow-y-auto'>
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
              <Dialog.Panel className='relative w-full transform overflow-hidden rounded-xl border border-gray-100 bg-white p-8 text-left shadow-xl shadow-gray-300/10 transition-all sm:my-8 sm:max-w-lg'>
                <AccessKeyDialogContent accessKey={accessKey} />
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition.Root>
  )
}

function ExpirationRateMenu({ selected, setSelected }) {
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
    <Listbox value={selected} onChange={setSelected}>
      <div className='relative'>
        <Listbox.Button
          ref={setReferenceElement}
          className='relative w-48 cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-10 text-left shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 sm:text-sm'
        >
          <span className='block truncate'>{selected.name}</span>
          <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
            <ChevronDownIcon
              className='-mr-1 ml-2 h-3 w-3'
              aria-hidden='true'
            />
          </span>
        </Listbox.Button>

        {ReactDOM.createPortal(
          <Listbox.Options
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className='absolute z-10 w-48 overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'
          >
            <div className='max-h-64 overflow-auto'>
              {EXPIRATION_RATE.map(rate => (
                <Listbox.Option
                  key={`${rate.name}-${rate.value}`}
                  className={({ active }) =>
                    `${
                      active ? 'bg-gray-100' : ''
                    } select-none py-2 px-3 hover:cursor-pointer`
                  }
                  value={rate}
                >
                  {({ selected }) => (
                    <div className='flex flex-row'>
                      <div className='flex flex-1 flex-col'>
                        <div className='flex justify-between py-0.5 font-medium'>
                          {rate.name}
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
          </Listbox.Options>,
          document.querySelector('body')
        )}
      </div>
    </Listbox>
  )
}

function CalendarInput({ setSelectedExpiry, selectedExpiry }) {
  const selectedCustom = moment()
    .add(
      parseInt(selectedExpiry.value),
      selectedExpiry.value.charAt(selectedExpiry.value.length - 1)
    )
    .format('YYYY/MM/DD')

  return (
    <Popover className='relative'>
      <Popover.Button className='relative w-48 cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-10 text-left shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 sm:text-sm'>
        {selectedCustom}
      </Popover.Button>
      <Popover.Panel className='absolute z-20 mt-2'>
        {({ close }) => (
          <Calendar
            selectedDate={selectedCustom}
            inactivityHour={720} // the default inactivity timeout is 30 days (720h)
            onChange={e => {
              const duration = moment
                .duration(
                  moment(e, 'YYYY/MM/DD')
                    .startOf('day')
                    .diff(moment().startOf('day'))
                )
                .asHours()

              setSelectedExpiry({
                name: CUSTOM_TITLE,
                value: duration + 'h',
                custom: true,
              })

              close()
            }}
          />
        )}
      </Popover.Panel>
    </Popover>
  )
}

export default function AccessKey() {
  const [selectedExpiry, setSelectedExpiry] = useState(EXPIRATION_RATE[0])
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [generatedAccessKey, setGeneratedAccessKey] = useState('')
  const [accessKeyDialogOpen, setAccessKeyDialogOpen] = useState(false)
  const router = useRouter()

  const { connector } = router.query

  const { user } = useUser()

  const { data: { items: connectors } = {} } = useSWR(
    '/api/users?name=connector&showSystem=true'
  )

  const userID = connector ? connectors?.[0]?.id : user.id

  useEffect(() => {
    setAccessKeyDialogOpen(generatedAccessKey.length > 0)
  }, [generatedAccessKey])

  async function onSubmit(e) {
    e.preventDefault()

    setError('')

    try {
      const res = await fetch('/api/access-keys', {
        method: 'POST',
        body: JSON.stringify({
          name,
          userID,
          expiry: selectedExpiry.value,
          inactivityTimeout: '720h',
        }),
      })

      const data = await jsonBody(res)

      setGeneratedAccessKey(data.accessKey)
    } catch (e) {
      setError(e.message)
    }

    return false
  }

  return (
    <div className='mx-auto w-full max-w-2xl'>
      <Head>
        <title>Add Access Key - Infra</title>
      </Head>
      <div className='flex items-center justify-between'>
        <h1 className='my-6 py-1 font-display text-xl font-medium'>
          {connector ? 'Create Connector Key' : 'Create Personal Key'}
        </h1>
        <Link href='/settings'>
          <XMarkIcon
            className='h-5 w-5 text-gray-500 hover:text-gray-800'
            aria-hidden='true'
          />
        </Link>
      </div>

      <div className='flex w-full flex-col'>
        <form onSubmit={onSubmit} className='mb-6 flex flex-col space-y-6'>
          <div className='flex flex-col space-y-1'>
            <label className='text-2xs font-medium text-gray-700'>Name</label>
            <input
              name='name'
              required
              autoFocus
              spellCheck='false'
              type='search'
              onKeyDown={e => {
                if (e.key === 'Escape' || e.key === 'Esc') {
                  e.preventDefault()
                }
              }}
              value={name}
              onChange={e => {
                setName(e.target.value)
                setError('')
              }}
              className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                error ? 'border-red-500' : 'border-gray-300'
              }`}
            />
          </div>
          <div className='flex flex-col space-y-1'>
            <label className='text-2xs font-medium text-gray-700'>
              Expiration
            </label>
            <div className='flex flex-col sm:flex-row sm:items-center'>
              <ExpirationRateMenu
                selected={selectedExpiry}
                setSelected={setSelectedExpiry}
              />
              {selectedExpiry.custom && (
                <div className='mt-4 sm:ml-4 sm:mt-0'>
                  <CalendarInput
                    selectedExpiry={selectedExpiry}
                    setSelectedExpiry={setSelectedExpiry}
                  />
                </div>
              )}
            </div>
          </div>
          {selectedExpiry?.value && (
            <div className='space-y-1 pt-6 text-xs text-gray-500'>
              {selectedExpiry.value === EXPIRATION_RATE[0].value ? (
                <div>
                  This access key will expire on{' '}
                  <span className='font-semibold text-gray-900'>
                    {moment()
                      .add(
                        parseInt(selectedExpiry.value),
                        selectedExpiry.value.charAt(
                          selectedExpiry.value.length - 1
                        )
                      )
                      .format('h:mm:ss a, MMMM Do YYYY')}
                  </span>
                </div>
              ) : (
                <>
                  <div>
                    This access key must be used at least once every{' '}
                    <span className='font-semibold text-gray-900'>30 days</span>
                    ,
                  </div>
                  <div>
                    and will expire on{' '}
                    <span className='font-semibold text-gray-900'>
                      {moment()
                        .add(
                          parseInt(selectedExpiry?.value),
                          selectedExpiry?.value?.charAt(
                            selectedExpiry?.value.length - 1
                          )
                        )
                        .format('h:mm:ss a, MMMM Do YYYY')}
                    </span>
                  </div>
                </>
              )}
            </div>
          )}
          {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
          <div className='flex items-center justify-end space-x-3 pt-5 pb-3'>
            <button
              type='submit'
              disabled={!name}
              className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-black'
            >
              Generate Access Key
            </button>
          </div>
          <AccessKeyDialog
            accessKeyDialogOpen={accessKeyDialogOpen}
            accessKey={generatedAccessKey}
          />
        </form>
      </div>
    </div>
  )
}

AccessKey.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
