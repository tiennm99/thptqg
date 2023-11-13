plugins {
    id("java")
}

group = "dev.miti99"
version = "1.0-SNAPSHOT"

repositories {
    mavenCentral()
}

dependencies {
    annotationProcessor("org.projectlombok:lombok:1.18.30")
    compileOnly("org.projectlombok:lombok:1.18.30")
    implementation("org.apache.logging.log4j:log4j-core:2.21.1")
    implementation("org.apache.logging.log4j:log4j-slf4j2-impl:2.21.1")
    implementation("org.apache.poi:poi:5.2.4")
    implementation("org.apache.poi:poi-ooxml:5.2.4")
    implementation("org.hibernate.orm:hibernate-community-dialects:6.3.1.Final")
    implementation("org.hibernate.orm:hibernate-core:6.3.1.Final")
    implementation("org.slf4j:slf4j-api:2.0.9")
    implementation("org.xerial:sqlite-jdbc:3.43.2.2")
}
