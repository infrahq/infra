import { PlusIcon } from '@heroicons/react/solid'
import Head from 'next/head'

import Layout from '../components/Layout'

export default function Index () {
	return (
		<Layout>
			<Head>
				<title>Infrastructure – Infra</title>
				<meta property="og:title" content="Infrastructure - Infra" key="title" />
			</Head>
			<div className="max-w-7xl mx-auto px-6 sm:px-8 md:px-10">
				<div className="text-center mt-32">
					<h3 className="my-4 text-3xl font-bold text-black">No clusters</h3>
					<p className="mt-1 text-lg text-gray-700">Get started by connecting your Kubernetes cluster.</p>
					<div className="mt-6">
					<button
						type="button"
						className="inline-flex items-center px-6 py-2 border border-transparent shadow-sm font-medium rounded-lg text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
					>
						<PlusIcon className="-ml-1 mr-2 h-4 w-4" aria-hidden="true" />
						Connect cluster
					</button>
					</div>
				</div>
			</div>
		</Layout>
	)
}
