use std::env;
use std::process;

fn main() {
    let args: Vec<String> = env::args().collect();

    if args.len() < 2 {
        print_usage();
        process::exit(1);
    }

    match args[1].as_str() {
        "ping" => {
            println!("pong");
        }
        "info" => {
            println!(
                "{{\"name\":\"media-worker\",\"version\":\"0.1.0\",\"arch\":\"{}\",\"os\":\"{}\"}}",
                env::consts::ARCH,
                env::consts::OS
            );
        }
        "validate-magnet" => {
            if args.len() < 3 {
                eprintln!("Error: URL magnet dibutuhkan.");
                process::exit(1);
            }
            let magnet = &args[2];
            let is_valid = magnet.starts_with("magnet:?") && magnet.contains("xt=urn:btih:");
            println!(
                "{{\"valid\":{},\"uri\":\"{}\"}}",
                is_valid,
                magnet.replace('\"', "\\\"")
            );
        }
        _ => {
            eprintln!("Error: Command '{}' tidak dikenali.", args[1]);
            print_usage();
            process::exit(1);
        }
    }
}

fn print_usage() {
    println!("Cinembrot Media Worker (Rust)");
    println!("Penggunaan:");
    println!("  media-worker ping               - Tes respons worker");
    println!("  media-worker info               - Informasi arsitektur & versi worker");
    println!("  media-worker validate-magnet <uri> - Validasi struktur URI magnet BitTorrent");
}

#[cfg(test)]
mod tests {
    #[test]
    fn test_magnet_validation() {
        let valid_magnet = "magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a";
        let invalid_magnet = "https://example.com/movie.mp4";

        assert!(valid_magnet.starts_with("magnet:?") && valid_magnet.contains("xt=urn:btih:"));
        assert!(!(invalid_magnet.starts_with("magnet:?") && invalid_magnet.contains("xt=urn:btih:")));
    }
}

