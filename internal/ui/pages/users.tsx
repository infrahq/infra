import { Transition, Dialog } from '@headlessui/react'
import { Fragment, useEffect, useState } from 'react'
import { ExclamationIcon } from '@heroicons/react/outline'
import dayjs from 'dayjs'
import { PlusIcon } from '@heroicons/react/solid'
import relativeTime from 'dayjs/plugin/relativeTime'
import Head from 'next/head'
import { V1, User } from '../gen/v1.pb'

import Layout from '../components/Layout'

dayjs.extend(relativeTime)

function DeleteModal ({ open, setOpen }: { open: boolean, setOpen: (open: boolean) => void }) {
	function handleDelete() {
		setOpen(false)
	}

	return (
		<Transition.Root show={open} as={Fragment}>
		<Dialog
			as="div"
			static
			className="fixed z-10 inset-0 overflow-y-auto"
			open={open}
			onClose={setOpen}
		>
			<div className="flex justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
				<Transition.Child
					as={Fragment}
					enter="ease-out duration-300"
					enterFrom="opacity-0"
					enterTo="opacity-100"
					leave="ease-in duration-200"
					leaveFrom="opacity-100"
					leaveTo="opacity-0"
				>
					<Dialog.Overlay className="fixed inset-0 bg-gray-100 bg-opacity-75 transition-opacity" />
				</Transition.Child>

				{/* This element is to trick the browser into centering the modal contents. */}
				<span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">
					&#8203;
				</span>
				<Transition.Child
					as={Fragment}
					enter="ease-out duration-300"
					enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
					enterTo="opacity-100 translate-y-0 sm:scale-100"
					leave="ease-in duration-200"
					leaveFrom="opacity-100 translate-y-0 sm:scale-100"
					leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
				>
					<div className="inline-block align-bottom bg-white rounded px-4 pt-5 pb-4 text-left overflow-hidden transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full sm:p-6">
						<div className="sm:flex sm:items-start">
							<div className="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-red-100 sm:mx-0 sm:h-10 sm:w-10">
								<ExclamationIcon className="h-6 w-6 text-red-600" aria-hidden="true" />
							</div>
							<div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
								<Dialog.Title as="h3" className="text-lg leading-6 font-medium text-gray-900">
									Delete User
								</Dialog.Title>
								<div className="mt-2">
									<p className="text-sm text-gray-500">
										Are you sure you want to delete this user? They will no longer have access. This action cannot be undone.
									</p>
								</div>
							</div>
						</div>
						<div className="mt-5 sm:mt-4 sm:flex sm:flex-row-reverse">
							<button
								type="button"
								className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-red-600 text-base font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:ml-3 sm:w-auto sm:text-sm"
								onClick={handleDelete}
							>
								Delete
							</button>
							<button
								type="button"
								className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:mt-0 sm:w-auto sm:text-sm"
								onClick={() => setOpen(false)}
							>
								Cancel
							</button>
						</div>
					</div>
				</Transition.Child>
			</div>
		</Dialog>
		</Transition.Root>
	)
}

function AddModal ({ open, setOpen, onUserAdded }: { open: boolean, setOpen: (open: boolean) => void, onUserAdded: () => void }) {
	const [email, setEmail] = useState('')
	const [password, setPassword] = useState('')

	async function handleSubmit (e: React.SyntheticEvent) {
		e.preventDefault()
		try {
			await V1.CreateUser({ email, password })
			setOpen(false)
			onUserAdded()
		} catch (e) {
			console.error(e)
		}
	}

	return (
		<Transition.Root show={open} as={Fragment}>
		<Dialog
			as="div"
			static
			className="fixed z-10 inset-0 overflow-y-auto"
			open={open}
			onClose={setOpen}
		>
			<div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-8 text-center sm:block sm:p-0">
				<Transition.Child
					as={Fragment}
					enter="ease-out duration-300"
					enterFrom="opacity-0"
					enterTo="opacity-100"
					leave="ease-in duration-200"
					leaveFrom="opacity-100"
					leaveTo="opacity-0"
				>
					<Dialog.Overlay className="fixed inset-0 bg-gray-100 bg-opacity-75 transition-opacity" />
				</Transition.Child>
				{/* This element is to trick the browser into centering the modal contents. */}
				<span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">
					&#8203;
				</span>
				<Transition.Child
					as={Fragment}
					enter="ease-out duration-300"
					enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
					enterTo="opacity-100 translate-y-0 sm:scale-100"
					leave="ease-in duration-200"
					leaveFrom="opacity-100 translate-y-0 sm:scale-100"
					leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
				>
					<div className="inline-block align-bottom bg-white rounded-xl px-6 py-5 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-md w-full sm:px-8 sm:py-4">
						<div className="">
							<div className="mt-3">
								<Dialog.Title as="h3" className="text-lg leading-6 font-medium text-gray-900">
									Add User
								</Dialog.Title>
								<div className="mt-2">
									<form onSubmit={handleSubmit} action="#" method="POST">
										<label htmlFor="email" className="block text-sm font-medium text-gray-700 mt-4">
											Email
										</label>
										<div className="mt-1">
											<input
												type="text"
												name="email"
												id="email"
												className="px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 block w-full border border-gray-300 rounded-lg"
												onChange={e => setEmail(e.target.value)}
											/>
										</div>
										<label htmlFor="password" className="block text-sm font-medium text-gray-700 mt-4">
											Password
										</label>
										<div className="mt-1">
											<input
												type="password"
												name="password"
												id="email"
												className="px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 block w-full border border-gray-300 rounded-lg"
												onChange={e => setPassword(e.target.value)}
											/>
										</div>
										<div className="mt-8 sm:flex sm:flex-row-reverse">
											<button
												type="submit"
												className="w-full inline-flex justify-center rounded-lg border border-transparent shadow-sm px-5 py-2 bg-blue-600 text-base font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:ml-3 sm:w-auto"
											>
												Add User
											</button>
											<button
												type="button"
												className="mt-3 w-full inline-flex justify-center rounded-lg border border-gray-300 shadow-sm px-5 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:mt-0 sm:w-auto"
												onClick={() => setOpen(false)}
											>
												Cancel
											</button>
										</div>
									</form>
								</div>
							</div>
						</div>
					</div>
				</Transition.Child>
			</div>
		</Dialog>
		</Transition.Root>
	)
}


function Table ({ users }: { users: User[] }) {
	const [deleteModalOpen, setDeleteModalOpen] = useState(false)

	return (
		<div className="flex flex-col">
			<div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
				<div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
				<div>
					<table className="min-w-full divide-y divide-gray-200">
						<thead className="bg-white">
							<tr>
							<th
								scope="col"
								className="px-1 pb-1.5 text-left text-sm font-semibold text-black"
							>
								Email
							</th>
							<th
								scope="col"
								className="px-1 pb-1.5 text-left text-sm font-semibold text-black"
							>
								Created
							</th>
							{/* <th
								scope="col"
								className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
							>
								Provider
							</th>
							<th
								scope="col"
								className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
							>
								Permission
							</th>
							<th scope="col" className="relative px-6 py-3">
								<span className="sr-only">Edit</span>
							</th> */}
							</tr>
						</thead>
						<tbody className="bg-white divide-y divide-gray-200">
							{users.map(user => (
								<tr key={user.id} className="hover:bg-gray-50 cursor-pointer">
									<td className="px-1 py-3 whitespace-nowrap text-gray-900">{user.email}{user.admin ? <span className="inline-flex items-center ml-2 px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">Admin</span> : null}</td>
									<td className="px-1 py-3 whitespace-nowrap text-gray-500">{dayjs(Number(user.created) * 1000).fromNow()}</td>
									{/* <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{user.provider}</td> */}
									{/* <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{user.permission.name}</td> */}
									{/* <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
									<a disabled={true} href="#" className="text-blue-600 hover:text-blue-900 disabled:text-black" onClick={() => setOpen(true)}>
										Delete
									</a>
									</td> */}
								</tr>
							))}
						</tbody>
					</table>
				</div>
				</div>
			</div>
			<DeleteModal open={deleteModalOpen} setOpen={setDeleteModalOpen} />
		</div>
	)
}

export default function Users () {
	const [users, setUsers] = useState([] as User[])
	const [addModalOpen, setAddModalOpen] = useState(false)

	async function fetchUsers () {
		let res = await V1.ListUsers({})
		setUsers(res.users || [])
	}

    useEffect(() => {
        fetchUsers()
    }, [])

	return (
		<Layout>
			<Head>
				<title>Users – Infra</title>
				<meta property="og:title" content="My page title" key="title" />
			</Head>
			<div className="mt-4">
				<div className="max-w-7xl mx-auto mb-6 px-4 sm:px-6 md:px-8 flex justify-between">
					<h1 className="text-3xl font-semibold text-gray-900">Users</h1>
					<div>
						<button
							type="button"
							className="inline-flex items-center px-5 py-1.5 border border-transparent font-medium rounded-lg text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
							onClick={() => { setAddModalOpen(true) }}
						>
							<PlusIcon className="-ml-1 mr-2 h-4 w-4" aria-hidden="true" />
							Add User
						</button>
					</div>
				</div>
				<div className="max-w-7xl mx-auto px-4 sm:px-6 md:px-8 mt-3">
					<Table users={users} />
				</div>
				<AddModal open={addModalOpen} setOpen={setAddModalOpen} onUserAdded={() => { fetchUsers() }} />
			</div>
		</Layout>
	)
}
