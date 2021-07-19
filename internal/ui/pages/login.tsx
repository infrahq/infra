import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import { useCookies } from 'react-cookie'
import { V1 } from '../gen/v1.pb'

export default function Login () {
	const router = useRouter()
	const [email, setEmail] = useState('')
	const [password, setPassword] = useState('')
	const [cookies] = useCookies(['login'])

    if (process.browser && cookies.login) {
        router.replace("/")
        return <></>
    }

	let nextParam = router.query.next ? router.query.next : "/"
	let next = "/"

	if (typeof nextParam === "string") {
		next = nextParam
	} else if (nextParam.length > 0) {
		next = nextParam[0]
	}

	const handleSubmit = useCallback(async e => {
		e.preventDefault()
		try {
			await V1.Login({ infra: { email, password } })
			router.replace(next)
		} catch (e) {
			console.error(e)
		}
	}, [email, password, router])

	useEffect(() => {
		router.prefetch(next)
	}, [])

	return (
		<div className="min-h-screen flex flex-col justify-center py-12 pb-48 sm:px-6 lg:px-8">
			<div className="sm:mx-auto sm:w-full sm:max-w-md">
				<img
					className="mx-auto h-7 w-auto text-blue-500 fill-current"
					src="/combo.svg"
					alt="Infra"
				/>
			</div>
			<div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
				<div className="bg-white pb-12 pt-10 px-4 sm:border sm:border-gray-200 sm:rounded-xl sm:px-10">
				<h2 className="mb-6 font-bold text-xl">Sign in to your account</h2>
				<form onSubmit={handleSubmit} className="space-y-6" action="#" method="POST">
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
								className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-lg placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
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
								className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-lg placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
								value={password}
								onChange={e => setPassword(e.target.value)}
							/>
						</div>
					</div>
					<div>
						<button
							type="submit"
							className="w-full flex justify-center mt-8 py-2.5 px-4 border border-transparent rounded-lg shadow-sm text-base font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
						>
							Sign in
						</button>
					</div>
				</form>
				</div>
			</div>
		</div>
	)
}
