import { useRouter } from "next/router";

export default function Index () {
    const router = useRouter()

    if (process.browser) {
        router.replace("/infrastructure")
    }

    return <></>
}
