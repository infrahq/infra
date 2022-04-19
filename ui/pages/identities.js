import useSWR from 'swr'
import { useTable } from 'react-table'
import { Fragment, useRef, useState } from 'react'
import dayjs from 'dayjs'
import { Dialog, Transition } from '@headlessui/react'
import { XIcon, PlusIcon } from '@heroicons/react/outline'

import Dashboard from '../components/dashboard'
import Table from '../components/table'
import DeleteModal from '../components/modals/delete'

const columns = [
  {
    Header: 'Identity',
    accessor: 'name',
    Cell: ({ value }) => (
      <div className='flex items-center'>
        <div className='w-9 h-9 mr-4 bg-purple-100/10 font-bold rounded-lg flex items-center justify-center'>{value[0]?.toUpperCase()}</div>
        <div>{value}</div>
      </div>
    )
  },
  {
    Header: 'Kind',
    accessor: 'kind'
  },
  {
    id: 'last_seen',
    accessor: i => {
      return i.lastSeenAt ? dayjs(i.lastSeenAt).fromNow() : 'never'
    },
    Header: () => (
      <div className='text-left'>
        Last Seen
      </div>
    ),
    Cell: ({ value }) => (
      <div className='text-left'>
        {value}
      </div>
    )
  }, {
    id: 'delete',
    accessor: (r) => r,
    Cell: ({ value: identity }) => {
      const [open, setOpen] = useState(false)
      return (
        <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
          <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer'>
            <XIcon className='w-5 h-5 text-gray-500' />
          </button>
          <DeleteModal
            open={open}
            setOpen={setOpen}
            title='Delete Identity'
            message={(<>Are you sure you want to delete <span className='font-bold'>{identity.name}</span>? This action cannot be undone.</>)}
          />
        </div>
      )
    }
  }
]

function AddModal ({ open, setOpen }) {
  const nameInputref = useRef(null)
  const [name, setName] = useState('')

  function form () {
    async function onSubmit (e) {
      e.preventDefault()

      const res = await fetch('/v1/identities', {
        method: 'POST',
        body: JSON.stringify({
          name
        })
      })

      console.log(res)

      return false
    }

    return (
      <form onSubmit={onSubmit} className='flex flex-col space-y-4 mx-8'>
        <div className='flex flex-col w-full space-y-4 my-8'>
          <input onChange={e => setName(e.target.value)} ref={nameInputref} placeholder='user@example.com' className='bg-zinc-800 border border-zinc-700 text-base px-4 py-2 rounded-lg focus:outline-none focus:ring focus:ring-cyan-600' />
        </div>
        <div className='mt-5 flex items-center justify-end space-x-4'>
          <button
            type='button'
            className='mt-3 inline-flex justify-center rounded-md bg-zinc-800 border border-zinc-700 shadow-sm px-6 py-2 text-base font-medium text-gray-300 hover:bg-gray-zinc-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:col-start-1 sm:text-sm'
            onClick={() => setOpen(false)}
          >
            Cancel
          </button>
          <button type='submit' className='block bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-lg p-0.5 my-2'>
            <div className='bg-black rounded-lg flex justify-center items-center tracking-tight px-8 py-2 '>
              Add Identity
            </div>
          </button>
        </div>
      </form>
    )
  }

  return (
    <Transition.Root show={open} as={Fragment}>
      <Dialog as='div' className='fixed z-10 inset-0 overflow-y-auto' initialFocus={nameInputref} onClose={setOpen}>
        <div className='flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0'>
          <Transition.Child
            as={Fragment}
            enter='ease-out duration-300'
            enterFrom='opacity-0'
            enterTo='opacity-100'
            leave='ease-in duration-200'
            leaveFrom='opacity-100'
            leaveTo='opacity-0'
          >
            <Dialog.Overlay className='fixed inset-0 bg-black bg-opacity-75 transition-opacity' />
          </Transition.Child>

          {/* This element is to trick the browser into centering the modal contents. */}
          <span className='hidden sm:inline-block sm:align-middle sm:h-screen' aria-hidden='true'>
            &#8203;
          </span>
          <Transition.Child
            as={Fragment}
            enter='ease-out duration-300'
            enterFrom='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
            enterTo='opacity-100 translate-y-0 sm:scale-100'
            leave='ease-in duration-200'
            leaveFrom='opacity-100 translate-y-0 sm:scale-100'
            leaveTo='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
          >
            <div className='relative inline-block align-bottom bg-zinc-900 rounded-lg px-6 pt-4 pb-4 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full sm:p-6'>
              <h1 className='text-2xl text-white font-bold tracking-tight text-center mt-2 mb-6'>Add Identity</h1>
              {form()}
            </div>
          </Transition.Child>
        </div>
      </Dialog>
    </Transition.Root>
  )
}

export default function () {
  const { data, error } = useSWR('/v1/identities', { fallbackData: [] })
  const [open, setOpen] = useState(false)

  const identities = error ? [] : data
  const table = useTable({ columns, data: identities })

  return (
    <Dashboard>
      <div className='flex flex-col my-20'>
        <header className='flex items-center'>
          <h1 className='text-4xl flex-1 font-bold my-8'>Identities</h1>
          <button onClick={() => setOpen(true)} className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2'>
            <div className='bg-black rounded-full flex items-center tracking-tight px-4 py-2 '>
              <PlusIcon className='w-5 h-5 mr-2' />Add Identity
            </div>
          </button>
        </header>
        {error?.status
          ? <div className='my-20 text-center font-light text-gray-400 text-2xl'>{error?.info?.message}</div>
          : data.length === 0
            ? <p>No identities</p>
            : <Table {...table} />}
        <AddModal open={open} setOpen={setOpen} />
      </div>
    </Dashboard>
  )
}
