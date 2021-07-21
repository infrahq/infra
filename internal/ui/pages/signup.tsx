import { useCallback, useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import { useCookies } from 'react-cookie'
import { V1 } from '../gen/v1.pb'

export default function Signup () {
	const router = useRouter()
	const [email, setEmail] = useState('')
	const [password, setPassword] = useState('')
	const [cookies] = useCookies(['login'])

	const handleSubmit = useCallback(async e => {
		e.preventDefault()
		try {
			await V1.Signup({ email, password })
			router.replace("/")
		} catch (e) {
			console.error(e)
		}
	}, [email, password])

	if (process.browser && cookies.login) {
        router.replace("/")
        return <></>
    }

	return (
		<div className="min-h-screen flex flex-col justify-center py-8 pb-48 sm:px-6 lg:px-8">
			<Head>
				<title>Signup – Infra</title>
				<meta property="og:title" content="Signup - Infra" key="title" />
			</Head>
			<div className="sm:mx-auto sm:w-full select-none">
				<img
					className="mx-auto text-blue-500 fill-current w-10 h-10"
					src="/icon.svg"
					alt="Infra"
				/>
			</div>
			<div className="sm:mx-auto sm:w-full sm:max-w-sm bg-white pb-12 pt-10 px-4">
				<h2 className="text-center mb-3 font-medium tracking-tight text-xl">Welcome to Infra</h2>
				<h3 className="text-center text-sm text-gray-500">Get started by creating your admin account</h3>
				<form onSubmit={handleSubmit} className="space-y-5 my-10" action="#" method="POST">
					<div>
						<label htmlFor="email" className="block text-sm font-medium text-gray-700">
							Email
						</label>
						<div className="mt-1">
							<input
								id="email"
								name="email"
								type="email"
								autoFocus
								autoComplete="email"
								required
								className="appearance-none block w-full px-3 py-2 border text-sm border-gray-300 rounded-md placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 shadow-sm"
								value={email}
								onChange={e => setEmail(e.target.value)}
							/>
						</div>
					</div>
					<div>
						<label htmlFor="password" className="block text-sm font-medium text-gray-700">
							Password
						</label>
						<div className="mt-1">
							<input
								id="password"
								name="password"
								type="password"
								autoComplete="current-password"
								required
								className="appearance-none block w-full px-3 py-2 text-sm border border-gray-300 rounded-lg placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 shadow-sm"
								value={password}
								onChange={e => setPassword(e.target.value)}
							/>
						</div>
					</div>
					<div>
						<button
							type="submit"
							className="w-full flex justify-center mt-8 py-2.5 px-4 border border-transparent rounded-lg shadow-sm text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
						>
							Create your account
						</button>
					</div>
				</form>
			</div>
		</div>
	)
}
